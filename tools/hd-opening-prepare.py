"""準備已觀測的 OP-FULL 美術與來源；不補原始 DAT 的尾端。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

from PIL import Image, __version__


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000 and __version__ == '9.4.0'
    parser = argparse.ArgumentParser()
    parser.add_argument('--register', action='store_true')
    args = parser.parse_args()
    project = Path('/project')
    catalog = project / 'workplace/hd-prototype/catalog'
    history = catalog / 'remaining/history'
    output = history / 'art'
    assert history.stat().st_uid == 1000 and not output.exists()
    evidence_path = history / 'source-verification.json'
    evidence = json.loads(evidence_path.read_text())
    assert evidence['verified']
    record = evidence['full_copy_records'][-1]
    assert record['source'] == 'OP-FULL.OZM' and record['all_256000_pixels_equal'] and record['loader_alias_verified']
    original_root = catalog / 'remaining/source-boundary/OP-FULL-runtime'
    original = original_root / 'OP-FULL-runtime.idx'
    old_receipt = json.loads((original_root / 'receipt.json').read_text())
    assert sha(original) == old_receipt['indexed_sha256'] == record['indexed_sha256']
    native_path = history / 'probe/opening-samples' / record['native_after']
    assert native_path.read_bytes() == original.read_bytes()
    generation_path = catalog / 'art/opening-full-generation-v1.json'
    generation = json.loads(generation_path.read_text())['records'][0]
    raw = catalog / 'art' / generation['asset']
    prompt = catalog / 'art' / generation['prompt']
    assert sha(raw) == generation['sha256'] and sha(prompt) == generation['prompt_sha256']
    for reference in generation['inputs']:
        assert sha(project / reference['path']) == reference['sha256']
    image = Image.open(raw)
    assert image.mode in ['RGB', 'RGBA'] and image.width >= 1280 and image.height >= 800
    assert abs(image.width / image.height - 1.6) < 0.02
    if image.mode == 'RGBA':
        assert image.getchannel('A').getextrema() == (255, 255)
    output.mkdir()
    prepared = output / 'OP-FULL.OZM-hd-v1.png'
    image.convert('RGB').resize((1280, 800), Image.Resampling.LANCZOS).save(prepared)
    (output / 'prepare-source.txt').write_bytes(Path(__file__).read_bytes())
    profile = {'source': 'OP-FULL.OZM', 'status': 'runtime_observed', 'width': 640, 'height': 400,
               'indices_sha256': sha(original), 'plane_sha256': record['plane_sha256'],
               'asset': 'opening-full/' + prepared.name, 'asset_sha256': sha(prepared),
               'raw_asset': generation['asset'], 'raw_asset_sha256': sha(raw),
               'prompt': generation['prompt'], 'prompt_sha256': sha(prompt),
               'source_proof_sha256': sha(evidence_path),
               'source_scope': 'two matching ordinary history executions; on-disk decode remains unsupported',
               'geometry': 'whole canvas 640x400 to 1280x800; no source bytes or text modified'}
    result = {'classification': 'calibrated generated art and exact runtime-observed source; not static decode completion',
              'tool_sha256': sha(Path(__file__)), 'generation_sha256': sha(generation_path),
              'pillow': __version__, 'record': profile}
    (output / 'profiles.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    if args.register:
        assert '歷史簡介完整複製路徑，READY' in (project / 'RESEARCH-LOG.md').read_text()
        decoded = catalog / 'decoded'
        assert decoded.stat().st_uid == 1000
        source_target = decoded / 'OP-FULL.OZM.idx'
        if source_target.exists():
            assert sha(source_target) == profile['indices_sha256']
        else:
            source_target.write_bytes(original.read_bytes())
        sidecar = decoded / 'OP-FULL.OZM-runtime-source.json'
        assert not sidecar.exists()
        sidecar.write_text(json.dumps({'classification': result['classification'],
                                      'source_proof_sha256': sha(evidence_path),
                                      'record': profile}, ensure_ascii=False, indent=2) + '\n')
        source = 'package main\n\n// Two exact ordinary RAM observations; original DAT tail is unchanged.\n'
        source += 'var hdGraphicsOpeningProfiles = []hdGraphicsProfile{{\n'
        source += 'Source: graphicsSourceProfile{Name:"OP-FULL.OZM",Status:"runtime_observed",Width:640,Height:400,Planes:[]string{'
        source += ','.join(json.dumps(value) for value in profile['plane_sha256']) + '}},\n'
        for field, key in [('IndicesSHA', 'indices_sha256'), ('Asset', 'asset'), ('AssetSHA', 'asset_sha256')]:
            source += field + ':' + json.dumps(profile[key]) + ',\n'
        source += '}}\n'
        go_path = project / 'tools/hd-graphics-opening-profiles.go'
        assert not go_path.exists()
        go_path.write_text(source)
        subprocess.run(['gofmt', '-w', str(go_path)], check=True)
    print(json.dumps({'prepared': str(prepared.relative_to(project)), 'registered': args.register,
                      'source_indices_sha256': profile['indices_sha256'], 'asset_sha256': profile['asset_sha256']}))


if __name__ == '__main__':
    main()
