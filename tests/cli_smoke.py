from pathlib import Path
import os,json,subprocess,tempfile
root=Path(__file__).resolve().parents[1]
exe=root/'build/release/shadow21'
with tempfile.TemporaryDirectory() as tmp:
 env=dict(os.environ,XDG_CONFIG_HOME=tmp)
 def run(*args,success=True):
  p=subprocess.run([str(exe),*args],env=env,capture_output=True,text=True,timeout=30)
  assert (p.returncode==0)==success,(args,p.returncode,p.stdout,p.stderr)
  return p
 s=json.loads(run('preferences').stdout)
 assert s['tmux_enabled'] is False
 assert s['auto_ip'] is True
 run('preferences','set','--auto-ip=false')
 assert json.loads(run('preferences').stdout)['auto_ip'] is False
 assert json.loads(run('tmux','list').stdout)==[]
 assert json.loads(run('tmux','status').stdout)['phase']=='inactive'
 run('preferences','set','--tmux=true')
 run('tmux','start','shadow21-test',success=False)
 run('preferences','set','--mode','proxy','--tmux=false')
 run('tmux','start','shadow21-test',success=False)
 run('preferences','set','--mode','system')
 run('preferences','set','--endpoint','0.0.0.0:9050',success=False)
 assert json.loads(run('preferences').stdout)['endpoint']=='127.0.0.1:9050'
 run('preferences','set','--language','de')
 assert 'Befehle:' in run('--help').stdout
 assert 'Verwendung:' in run('tmux','--help').stdout
 assert 'Vorgang fehlgeschlagen' in run('tmux','start','other',success=False).stderr
 run('--lang','en','tmux','stop','other',success=False)
 run('preferences','set','--language','en')
 assert 'Commands:' in run('--help').stdout
 assert 'tmux' in run('--help').stdout and 'health' in run('--help').stdout
 assert 'shadow21' in run('--version').stdout
 run('mac','random','../eth0',success=False)
 assert 'MAC-Adresse' in run('--lang','de','mac','--help').stdout
print('PASS: CLI preferences, JSON, disabled tmux, input validation, DE/EN help and errors')
