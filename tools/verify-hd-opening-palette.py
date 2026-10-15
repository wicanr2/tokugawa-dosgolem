"""核對原版遊戲的兩種色盤變化，HD RGB 轉換另標近似。"""
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rgb6(value):
    return value << 2 | value >> 4


def cycle(target, steps):
    return [[rgb6((min(channel + steps, 21) if channel <= 21 else channel + steps) & 63)
             for channel in color] for color in target]


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project = Path('/project')
    history = project / 'workplace/hd-prototype/catalog/remaining/history'
    early_path = history / 'palette/verification-opening-palette.json'
    late_path = history / 'palette-late/verification-opening-palette-late.json'
    early, late = [json.loads(path.read_text()) for path in [early_path, late_path]]
    assert early['verified'] and late['verified']
    for proof, directory in [(early, history / 'palette'), (late, history / 'palette-late')]:
        assert proof['tool_sha256'] == sha(directory / 'verifier-source.txt')
        receipt = directory / f'desktop-{proof["mode"]}.json'
        assert proof['receipt_sha256'] == sha(receipt)
    op4 = [color[:3] for color in json.loads((project / 'workplace/hd-prototype/catalog/battle-frame/source-verification.json').read_text())['native_palette']]
    full = early['stages']['OP-FULL.OZM@542000000']['active_palette']
    bases = {'OP4.OZM': op4, 'OP-FULL.OZM': full}
    inverse = {rgb6(level): level for level in range(64)}
    records = {}
    for name, base in bases.items():
        target = [[0 if channel == 0 else inverse[channel] + 1 for channel in color] for color in base]
        assert all(0 <= channel <= 63 for color in target for channel in color)
        observed = []
        for key, sample in early['stages'].items():
            if key.split('@')[0] != name:
                continue
            palette = sample['active_palette']
            limit = max(channel for color in palette for channel in color)
            assert limit in inverse
            assert palette == [[min(channel, limit) for channel in color] for color in base]
            observed.append({'step': sample['step'], 'native_limit_rgb8': limit, 'native_limit_dac6': inverse[limit]})
        assert len(observed) == 3
        records[name] = {'base_palette_rgb8': base, 'target_palette_dac6': target,
                         'clamp_samples': observed, 'cycle_samples': []}
    for key, sample in late['stages'].items():
        assert key.startswith('OP-FULL.OZM@')
        palette = sample['active_palette']
        steps = inverse[palette[0][0]]
        assert steps in range(22)
        assert palette == cycle(records['OP-FULL.OZM']['target_palette_dac6'], steps)
        records['OP-FULL.OZM']['cycle_samples'].append({'step': sample['step'], 'cycle_steps': steps,
                                                       'palette_rgb8': palette})
    assert [sample['cycle_steps'] for sample in records['OP-FULL.OZM']['cycle_samples']] == [14, 20]
    ida_path = history / 'ida/range-1bbac-1be80.json'
    ida = json.loads(ida_path.read_text())
    assert ida['version'] == '9.4' and ida['input_sha256'] == '2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a'
    rows = {int(row['ida_linear_ea'], 16): row for row in ida['rows']}
    for address, raw in [(0x1bc6e, 'fec0'), (0x1bc70, '3ae0'), (0x1be28, '3c15')]:
        assert rows[address]['bytes'] == raw
    result = {'verified': True, 'tool_sha256': sha(Path(__file__)), 'ida_evidence_sha256': sha(ida_path),
              'address_space': 'IDA LE loader linear EA; file offsets retained in cited export',
              'early_proof_sha256': sha(early_path), 'late_proof_sha256': sha(late_path),
              'native_contract': 'confirmed against all 48 palette channels in six clamp and two cycle samples',
              'records': records,
              'HD_transfer_level': 'presentation approximation for new 24-bit artwork; does not change guest palette or timing',
              'HD_clamp_formula': 'min(HD_rgb8, floor(native_limit_rgb8 * 255 / max(base_palette_rgb8)))',
              'HD_cycle_formula': 'target6=0 if HD8==0 else min(63,(HD8>>2)+1); cycle as native; rgb8=(v6<<2)|(v6>>4)',
              'limitations': ['only these two exact sources accept the verified effect families',
                              'unknown palettes continue to use native rendering; not full-game completion']}
    path = history / 'palette-effects.json'
    assert not path.exists()
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    (history / 'palette-effects-source.txt').write_bytes(Path(__file__).read_bytes())
    print(json.dumps({'verified': True, 'clamp_samples': 6, 'cycle_samples': 2, 'summary_sha256': sha(path)}))


if __name__ == '__main__':
    main()
