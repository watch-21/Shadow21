from pathlib import Path
import os, subprocess, tempfile, shutil
root = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory() as tmp:
    base = Path(tmp)
    (base / 'gui').mkdir()
    (base / 'release').mkdir()
    runner = base / 'gui/test'
    shutil.copy2(root / 'build/gui/shadow21-telemetry-test', runner)
    phase = base / 'phase'
    phase.write_text('inactive')
    cli = base / 'release/shadow21'
    cli.write_text('''#!/usr/bin/python3
import sys,json,os,time
from pathlib import Path
args=sys.argv[1:]
if args and args[0]=='preferences':
 p=Path(os.environ['TELEMETRY_PHASE']+'.auto')
 if '--auto-ip=false' in args:p.write_text('false')
 print(json.dumps({'language':'en','mode':'system','auto_ip':not p.exists()}))
elif args==['health']:print('{}')
elif args==['tmux','status']:print(json.dumps({'phase':Path(os.environ['TELEMETRY_PHASE']).read_text(),'mode':'proxy','owner':os.getuid(),'sessions':['tor']}))
elif args in [['ip','--json'],['proxy','test']]:
 p=Path(os.environ['TELEMETRY_PHASE']+'.'+('proxy' if args[0]=='proxy' else 'system'))
 n=int(p.read_text())+1 if p.exists() else 1;p.write_text(str(n));time.sleep(0.03)
 print(json.dumps({'ip':'198.51.100.42' if args[0]=='proxy' else '203.0.113.1','is_tor':args[0]=='proxy','sequence':n}))
else:sys.exit(2)
''')
    cli.chmod(0o755)
    p = subprocess.run([str(runner)], env=dict(os.environ, TELEMETRY_PHASE=str(phase), XDG_CONFIG_HOME=tmp), capture_output=True, text=True, timeout=20)
    assert p.returncode == 0, (p.stdout, p.stderr)
    print(p.stdout.strip())
