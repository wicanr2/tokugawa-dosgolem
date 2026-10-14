"""獨立核對共同 OZM 繪圖診斷；原版資料與執行收據僅留本機。"""
import hashlib
import json
import os
from collections import Counter
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rect(value):
    return tuple(value[k][a] for k, a in [('Min', 'X'), ('Min', 'Y'), ('Max', 'X'), ('Max', 'Y')])


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, out = Path('/project'), Path('/output')
    probe = out/'source-probe'
    receipt_path = probe/'startup-probe/latest.json'
    receipt = json.loads(receipt_path.read_text())
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    assert sha(baseline_path) == '1eb3fe503bac5ecfb0261e0f94cf762a765895d5d4fb9b4a0e1e712ee210ac7c'
    baseline = json.loads(baseline_path.read_text())
    fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip',
              'trace_tail', 'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame',
              'cd_pcm', 'mounted_disc', 'mouse_input_events', 'input_boundary_frames',
              'real_mode_history', 'real_mode_last']
    for key in fields:
        assert receipt[key] == baseline[key], key
    assert receipt['successful_steps'] == 750000000 and len(receipt['mouse_input_events']) == 390
    assert receipt['fd2_startup_calls'] == 0 and receipt['graphics_diagnostic_enabled']
    ram_sha = sha(probe/'startup-memory.bin')
    assert ram_sha == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(probe/'source-snapshot'/name) == digest, name
    catalog = project/'workplace/hd-prototype/catalog'
    inventory_path = catalog/'decoded/inventory.json'
    sources = {r['name']:r for r in json.loads(inventory_path.read_text())['records']}
    trace = receipt['graphics_draw_trace']
    assert trace['dropped_events'] == 0
    records = []
    for draw in trace['records']:
        if not draw['verified']:
            assert draw.get('observation_error'), draw
            continue
        source = sources[draw['resource']]
        path = catalog/'decoded'/f"{source['name']}.idx"
        assert sha(path) == source['indices_sha256']
        x0, y0, x1, y1 = rect(draw['SourceRect'])
        dx0, dy0, dx1, dy1 = rect(draw['DestRect'])
        width, height = source['width'], source['height']
        assert 0 <= x0 < x1 <= width and 0 <= y0 < y1 <= height
        assert 0 <= dx0 < dx1 <= 640 and 0 <= dy0 < dy1 <= 400
        assert (x1-x0, y1-y0) == (dx1-dx0, dy1-dy0)
        offset, columns, rows, pitch, dest = draw['raw_args']
        assert (x0, y0) == (offset % 80 * 8, offset // 80)
        assert (dx0, dy0) == ((dest-0xa0000) % 80 * 8, (dest-0xa0000) // 80)
        assert (columns*8, rows, pitch) == (x1-x0, y1-y0, 80)
        idx = path.read_bytes()
        crop = b''.join(idx[y*width+x0:y*width+x1] for y in range(y0, y1))
        digest = hashlib.sha256(crop).hexdigest()
        assert draw['ExpectedSHA'] == draw['ActualSHA'] == digest
        assert draw['BeginStep'] < draw['EndStep'] and not draw.get('observation_error')
        records.append({'source':source['name'], 'source_indices_sha256':sha(path),
                        'source_rect':[x0,y0,x1,y1], 'destination_rect':[dx0,dy0,dx1,dy1],
                        'begin_step':draw['BeginStep'], 'natural_return_step':draw['EndStep'],
                        'indexed_crop_sha256':digest})
    assert any(r['source'] == 'WEATHER1.OZM' and r['destination_rect'] == [464,291,608,365] for r in records)
    result = {'classification':'read-only original normal-path OZM opaque draw evidence',
              'verified':True, 'receipt_sha256':sha(receipt_path), 'baseline_sha256':sha(baseline_path),
              'complete_RAM_sha256':ram_sha, 'unchanged_fields':fields,
              'inventory_sha256':sha(inventory_path), 'tool_sha256':sha(Path(__file__)),
              'source_counts':dict(Counter(r['source'] for r in records)), 'records':records,
              'limitations':[r for r in trace['records'] if not r['verified']]}
    target = out/'source-verification.json'
    assert not target.exists(), 'preserve previous verification'
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified':True, 'source_counts':result['source_counts'],
                      'limitations':len(result['limitations']), 'sha256':sha(target)}, ensure_ascii=False))


if __name__ == '__main__':
    main()
