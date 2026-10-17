from pathlib import Path
import os,subprocess,tempfile
root=Path(__file__).resolve().parents[1];exe=root/'build/gui/shadow21-gui';cli=root/'build/release/shadow21'
out=root/'docs/ui';out.mkdir(exist_ok=True)
with tempfile.TemporaryDirectory() as tmp:
 env=dict(os.environ,XDG_CONFIG_HOME=tmp,QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software')
 for language in ['en','de']:
  subprocess.run([str(cli),'preferences','set','--language',language],env=env,check=True,capture_output=True)
  for label,options,mode in [('main',[],'system'),('compact',['--size','480x460'],'system'),('proxy-main',[],'proxy'),('tmux',['--tmux'],'proxy'),('tmux-start-dialog',['--tmux','--dialog','start'],'proxy'),('mac-dialog',['--dialog','mac'],'system'),('mac-restore-dialog',['--dialog','mac-restore'],'system'),('mac-compact-dialog',['--size','720x540','--dialog','mac-restore'],'system'),('large-main',['--size','1100x1000'],'system'),('large-tmux',['--tmux','--size','1100x1150'],'proxy')]:
   subprocess.run([str(cli),'preferences','set','--mode',mode],env=env,check=True,capture_output=True)
   path=out/f'{language}-{label}.png'
   p=subprocess.run([str(exe),'--inspect-ui',str(path),*options],env=env,capture_output=True,text=True,timeout=40)
   assert p.returncode==0 and path.is_file() and path.stat().st_size>20000,(language,label,p.stderr)
   for marker in ['QQmlApplicationEngine failed','ReferenceError','TypeError','Cannot assign','Binding loop','Unable to assign']:
    assert marker not in p.stderr,(language,label,p.stderr)
print('PASS: single-window main, compact, tmux, MAC confirmation and large EN/DE layouts rendered in English and German')
