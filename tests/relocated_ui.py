"""Load the embedded UI away from the build tree and its on-disk qmldir."""
from pathlib import Path
import os,subprocess,tempfile,shutil
root=Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory() as tmp:
    base=Path(tmp);(base/'bin').mkdir();(base/'release').mkdir()
    gui=base/'bin/shadow21-gui';cli=base/'release/shadow21'
    shutil.copy2(root/'build/gui/shadow21-gui',gui);shutil.copy2(root/'build/release/shadow21',cli)
    env=dict(os.environ,XDG_CONFIG_HOME=str(base/'config'),QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software')
    for language in ['en','de']:
        subprocess.run([str(cli),'preferences','set','--language',language],env=env,check=True,capture_output=True)
        for label,options in [('main',[]),('tmux',['--tmux'])]:
            target=base/(language+'-'+label+'.png')
            p=subprocess.run([str(gui),'--inspect-ui',str(target),*options],cwd=tmp,env=env,capture_output=True,text=True,timeout=30)
            assert p.returncode==0 and target.exists() and target.stat().st_size>20000,(language,label,p.stderr)
            for text in ['failed to load component','is not a type','ReferenceError','TypeError']:
                assert text not in p.stderr,(language,label,p.stderr)
print('PASS: relocated standalone GUI loads embedded main/tmux components in English and German without build-tree qmldir files')
