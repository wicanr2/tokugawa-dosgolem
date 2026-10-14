"""保留 OZM 真實 EOF 與 PCX 來源證據；不猜補不存在的原始 bytes。"""
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import struct
import sys

from PIL import Image, __version__


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    project=Path('/project')
    root=project/'workplace/hd-prototype/catalog/remaining/source-boundary'
    assert root.parent.stat().st_uid==1000 and not root.exists()
    root.mkdir()
    dat=(project/'original/game/DX5.DAT').read_bytes()
    inventory=json.loads((project/'workplace/hd-prototype/asset-inventory.json').read_text())
    by_name={r['name']:r for r in inventory['records']}
    decoder=project/'tools/hd-ozm-probe.py'
    spec=importlib.util.spec_from_file_location('original_ozm_decoder',decoder)
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    groups={}
    for name in ['B-6.OZM','B-7.OZM','C-10.OZM','E-4.OZM','OP-FULL.OZM']:
        record=by_name[name]
        payload=dat[record['candidate_file_offset']:record['candidate_file_offset']+record['candidate_length']]
        assert sha(payload)==record['payload_sha256']
        if record['payload_sha256'] in groups:
            groups[record['payload_sha256']]['aliases'].append(name)
            continue
        facts={}
        def trace(frame,event,arg):
            if frame.f_code.co_name=='decode' and event=='exception':
                local=frame.f_locals
                values=bytes(local.get('values',b''))
                facts.update(decoder_line=frame.f_lineno,position=local.get('position'),plane=local.get('plane'),
                             decoded_plane_bytes=len(values),goal=local.get('columns',0)*local.get('rows',0),
                             current_value=local.get('value'),previous_command=local.get('command'),
                             known_partial_column_order_sha256=sha(values),known_partial_tail_hex=values[-16:].hex(),
                             completed_plane_sha256=[sha(x) for x in local.get('planes',[])],
                             completed_plane_tails_hex=[bytes(x[-8:]).hex() for x in local.get('planes',[])])
            return trace
        sys.settrace(trace)
        try:module.decode(payload)
        except Exception as error:facts['error']=type(error).__name__+': '+str(error)
        finally:sys.settrace(None)
        assert facts['plane']==3 and facts['goal']-facts['decoded_plane_bytes']==1
        columns,rows=struct.unpack_from('<HH',payload)
        groups[record['payload_sha256']]={'source':name,'aliases':[name],'payload_sha256':record['payload_sha256'],
             'dat_file_offset':record['candidate_file_offset'],'payload_length':len(payload),'columns':columns,'rows':rows,
             'last_payload_bytes_hex':payload[-24:].hex(),'exception':facts,
             'source_line':decoder.read_text().splitlines()[facts['decoder_line']-1].strip(),
             'unknown_output_pixels':[columns*8-8,rows-1,columns*8,rows],
             'status':'partial known source only; missing final plane byte remains unknown'}
    record=by_name['RAINEND.PCX'];payload=dat[record['candidate_file_offset']:record['candidate_file_offset']+record['candidate_length']]
    assert sha(payload)==record['payload_sha256']
    image=Image.open(io.BytesIO(payload));image.load();assert image.format=='PCX'
    path=root/'RAINEND-original-reference.png';image.convert('RGB').save(path)
    result={'classification':'original format boundary diagnostic; no bytes filled and no runtime support claim',
            'address_space':'DAT file offsets and payload-relative offsets; not IDA or guest linear addresses',
            'dat_sha256':sha(dat),'decoder_sha256':sha(decoder.read_bytes()),'tool_sha256':sha(Path(__file__).read_bytes()),
            'python':sys.version.split()[0],'pillow':__version__,'OZM_groups':list(groups.values()),
            'PCX':{'source':'RAINEND.PCX','payload_sha256':sha(payload),'dat_file_offset':record['candidate_file_offset'],
                   'size':list(image.size),'mode':image.mode,'colors':len(image.getcolors(65536) or []),
                   'header_hex':payload[:128].hex(),'reference':path.name,'reference_sha256':sha(path.read_bytes()),
                   'status':'source decoded for local research; original caller and HD presentation unverified'}}
    (root/'facts.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    (root/'probe-source.txt').write_bytes(Path(__file__).read_bytes())
    print(json.dumps({'OZM_groups':len(groups),'OZM_resources':5,'unknown_plane_bytes_per_resource':1,
                      'PCX_size':list(image.size),'PCX_colors':len(image.getcolors(65536) or [])}))


if __name__=='__main__':main()
