"""Exercise migration refusal using only paths in a temporary filesystem tree."""
from pathlib import Path
import os,subprocess,tempfile
root=Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory() as tmp:
    base=Path(tmp);control=base/'control';tree=base/'root';binpath=base/'bin';binpath.mkdir()
    subprocess.run(['dpkg-deb','--control',str(root/'dist/shadow21_v1_amd64.deb'),str(control)],check=True)
    source=(control/'preinst').read_text()
    for prefix in ['/run/','/var/lib/','/usr/bin/','/usr/lib/']:
        source=source.replace(prefix,str(tree)+prefix)
    script=base/'preinst';script.write_text(source)
    (tree/'run/systemd/system').mkdir(parents=True)
    systemctl=binpath/'systemctl'
    systemctl.write_text('''#!/usr/bin/python3
import os,sys
if sys.argv[1]=='is-active':sys.exit(0 if sys.argv[-1]==os.environ.get('ACTIVE_UNIT') else 1)
sys.exit(0)
''');systemctl.chmod(0o755)
    env=dict(os.environ,PATH=str(binpath))
    def run(ok,**extra):
        p=subprocess.run(['/bin/sh',str(script),'install'],env=dict(env,**extra),capture_output=True,text=True)
        assert (p.returncode==0)==ok,(p.returncode,p.stdout,p.stderr)
    run(True)
    for path in ['usr/bin/wvpn21','var/lib/wvpn21/install-torrc.json','run/wvpn21/active','run/wvpn21-mode.json','var/lib/wvpn21/proxy-state.json','var/lib/shadow21/proxy-state.json']:
        p=tree/path;p.parent.mkdir(parents=True,exist_ok=True);p.write_text('fixture');p.chmod(0o755)
        run(False);p.unlink()
    run(False,ACTIVE_UNIT='wvpn21d.service')
    run(False,ACTIVE_UNIT='shadow21d.service')
    run(True)
print('PASS: package migration preflight refuses old executables, old Tor journals, old/new active state and active routing services (isolated paths/commands)')
