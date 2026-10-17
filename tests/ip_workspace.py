from pathlib import Path
import shutil,subprocess,tempfile,os
root=Path(__file__).resolve().parents[1]
exe=root/'build/gui/shadow21-workspace-test'
with tempfile.TemporaryDirectory() as tmp:
 base=Path(tmp);(base/'gui').mkdir();(base/'release').mkdir()
 runner=base/'gui/test';shutil.copy2(exe,runner)
 cli=base/'release/shadow21'
 cli.write_text('''#!/usr/bin/python3
import sys,time,json,os
from pathlib import Path
if sys.argv[1:]==['ip','--json']:
 time.sleep(0.4)
 print(json.dumps({'ip':'203.0.113.1','is_tor':False,'elapsed_ms':400}))
elif sys.argv[1:]==['proxy','test']:
 p=Path(os.environ['IP_TEST_COUNT']);n=int(p.read_text())+1 if p.exists() else 1;p.write_text(str(n))
 time.sleep(0.4 if n==5 else 0.03)
 ip={1:'198.51.100.42',2:'198.51.100.42',3:'198.51.100.43',4:'not-an-address',5:'198.51.100.99',6:'198.51.100.44'}[n]
 print(json.dumps({'ip':ip,'is_tor':True,'elapsed_ms':30}))
else:
 sys.exit(2)
''')
 cli.chmod(0o755)
 result=subprocess.run([str(runner)],env=dict(os.environ,IP_TEST_COUNT=str(base/'count')),capture_output=True,text=True,timeout=12)
 assert result.returncode==0,(result.stdout,result.stderr)
 print(result.stdout.strip())
