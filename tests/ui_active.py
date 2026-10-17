"""Render active route states with a local CLI fixture; no networking changes."""
from pathlib import Path
import os,subprocess,tempfile,shutil
root=Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory() as tmp:
 base=Path(tmp);(base/'bin').mkdir();(base/'release').mkdir()
 gui=base/'bin/shadow21-gui';cli=base/'release/shadow21'
 shutil.copy2(root/'build/gui/shadow21-gui',gui)
 cli.write_text('''#!/usr/bin/python3
import os,sys,json
mode=os.environ['UI_MODE'];lang=os.environ['UI_LANG'];args=sys.argv[1:]
if args==['preferences']:print(json.dumps({'mode':mode,'language':lang,'tmux_enabled':True,'auto_ip':False,'endpoint':'127.0.0.1:9050'}))
elif args==['status','--json']:print(json.dumps({'shadow21_active':mode=='system','tor_active':True,'dns_state':'tor' if mode=='system' else 'clearnet','dns_addresses':['127.0.0.1'] if mode=='system' else []}))
elif args==['tmux','status']:print(json.dumps({'phase':'active' if mode=='proxy' else 'inactive','mode':mode,'sessions':['0'] if mode=='proxy' else [],'owner':os.getuid()}))
elif args==['health']:print('{}')
else:sys.exit(2)
''');cli.chmod(0o755)
 for lang in ['en','de']:
  for mode in ['system','proxy']:
   env=dict(os.environ,UI_MODE=mode,UI_LANG=lang,XDG_CONFIG_HOME=str(base/'config'),QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software')
   out=root/'docs/ui'/f'{lang}-active-{mode}.png'
   p=subprocess.run([str(gui),'--inspect-ui',str(out),'--size','1100x1100'],env=env,cwd=tmp,capture_output=True,text=True,timeout=30)
   assert p.returncode==0 and out.exists() and out.stat().st_size>20000,(lang,mode,p.stderr)
   for marker in ['ReferenceError','TypeError','Binding loop','Unable to assign','Cannot assign']:assert marker not in p.stderr,(lang,mode,p.stderr)
print('PASS: active system/proxy layouts rendered in English and German with isolated CLI fixtures')
