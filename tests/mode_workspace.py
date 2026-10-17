"""Exercise mode completion paths without privileged networking."""
from pathlib import Path
import os,subprocess,tempfile,shutil
root=Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory() as tmp:
    base=Path(tmp);(base/'gui').mkdir();(base/'release').mkdir()
    runner=base/'gui/test';shutil.copy2(root/'build/gui/shadow21-mode-test',runner)
    cli=base/'release/shadow21'
    cli.write_text('''#!/usr/bin/python3
import sys,json
if sys.argv[1:3]==['preferences','set']:
 args=sys.argv[3:];mode=args[args.index('--mode')+1]
 print(json.dumps({'language':'en','mode':mode,'tmux_enabled':True,'auto_ip':False,'endpoint':'127.0.0.1:9050'}))
elif sys.argv[1:]==['tmux','status']:print('{"phase":"inactive","mode":"none"}')
elif sys.argv[1:]==['status','--json']:print('{"shadow21_active":false,"tor_active":false,"dns_state":"clearnet","dns_addresses":[]}')
elif sys.argv[1:]==['health']:print('{}')
else:sys.exit(2)
''');cli.chmod(0o755)
    p=subprocess.run([str(runner)],env=dict(os.environ,XDG_CONFIG_HOME=tmp),capture_output=True,text=True,timeout=20)
    assert p.returncode==0,(p.stdout,p.stderr)
    print(p.stdout.strip())
