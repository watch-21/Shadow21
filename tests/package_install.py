"""Exercise generated maintenance scripts with isolated command fixtures."""
from pathlib import Path
import json,os,subprocess,tempfile
root=Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory() as tmp:
 base=Path(tmp);control=base/'control';binpath=base/'bin';binpath.mkdir();systemd=base/'systemd';systemd.mkdir();log=base/'events.jsonl'
 subprocess.run(['dpkg-deb','--control',str(root/'dist/shadow21_v1_amd64.deb'),str(control)],check=True)
 metadata=(control/'control').read_text()
 assert 'libvterm0' not in metadata,'Obsolete embedded terminal dependency'
 assert 'xterm' in metadata,'External terminal dependency missing'
 assert 'qml6-module-qtquick-shapes' in metadata,'QtQuick.Shapes runtime dependency missing'
 helper=binpath/'helper'
 helper.write_text('''#!/usr/bin/python3
import os,sys,json,shutil
with open(os.environ['TEST_EVENTS'],'a') as f:f.write(json.dumps(['helper',*sys.argv[1:]])+'\\n')
if sys.argv[1:]==['install-proxy-config'] and not shutil.which('tor'):sys.exit(1)
''');helper.chmod(0o755)
 systemctl=binpath/'systemctl';systemctl.write_text('''#!/usr/bin/python3
import os,sys,json
with open(os.environ['TEST_EVENTS'],'a') as f:f.write(json.dumps(['systemctl',*sys.argv[1:]])+'\\n')
''');systemctl.chmod(0o755)
 for name in ['postinst','prerm']:
  p=control/name;p.write_text(p.read_text().replace('/usr/lib/shadow21/shadow21-helper',str(helper)).replace('/run/systemd/system',str(systemd)))
 env=dict(os.environ,PATH=str(binpath),TEST_EVENTS=str(log))
 p=subprocess.run(['/bin/sh',str(control/'postinst'),'configure'],env=env,capture_output=True,text=True)
 assert p.returncode!=0 and not log.exists(),'Configuration helper ran without Tor'
 tmux=binpath/'tmux';tmux.write_text('#!/bin/sh\nexit 0\n');tmux.chmod(0o755)
 tor=binpath/'tor';tor.write_text('#!/bin/sh\nexit 0\n');tor.chmod(0o755)
 def run(script,*args):
  log.unlink(missing_ok=True)
  p=subprocess.run(['/bin/sh',str(control/script),*args],env=env,capture_output=True,text=True)
  assert p.returncode==0,(script,args,p.stderr)
  return [json.loads(s) for s in log.read_text().splitlines()]
 tor.unlink()
 p=subprocess.run(['/bin/sh',str(control/'postinst'),'configure'],env=env,capture_output=True,text=True)
 assert p.returncode!=0 and not log.exists(),'Configuration helper ran without Tor when tmux exists'
 tor.write_text('#!/bin/sh\nexit 0\n');tor.chmod(0o755)
 tmux.unlink()
 p=subprocess.run(['/bin/sh',str(control/'postinst'),'configure'],env=env,capture_output=True,text=True)
 assert p.returncode!=0 and not log.exists(),'Configuration helper ran without tmux when Tor exists'
 tmux.write_text('#!/bin/sh\nexit 0\n');tmux.chmod(0o755)
 events=run('postinst','configure')
 assert events[0]==['helper','install-proxy-config']
 assert ['systemctl','daemon-reload'] in events and not any('restart' in x or 'reload' in x for x in events)
 events=run('prerm','upgrade','1.1.2')
 assert ['helper','recover'] in events and ['helper','uninstall-proxy-config'] not in events
 assert not any(x[:2]==['systemctl','disable'] for x in events),'Upgrade disabled saved startup preference'
 events=run('prerm','remove')
 assert events.index(['helper','recover'])<events.index(['helper','uninstall-proxy-config'])
 assert any(x[:2]==['systemctl','disable'] for x in events)
print('PASS: generated postinst refuses missing Tor or tmux, prepares SOCKS before desktop/service setup without restart, upgrade retains settings, removal restores install configuration after recovery (command fixtures only)')
