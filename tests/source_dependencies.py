"""Exercise only the dependency/configuration fragment in a disposable directory."""
from pathlib import Path
import os, shlex, subprocess, tempfile
root = Path(__file__).resolve().parents[1]
source = (root / 'scripts/install.sh').read_text()
fragment = source[source.index('missing=()'):source.index('install -d -m 755 /usr/lib/shadow21')]
with tempfile.TemporaryDirectory() as tmp:
    base = Path(tmp)
    binpath = base / 'bin'
    binpath.mkdir()
    (base / 'build/release').mkdir(parents=True)
    helper = base / 'build/release/shadow21-helper'
    helper.write_text('#!/bin/sh\nprintf "config\\n" >> "$TEST_LOG"\n')
    helper.chmod(0o755)
    for present in [(), ('tor',), ('tmux',), ('tor', 'tmux')]:
        for n in ['tor', 'tmux']:
            p = binpath / n
            p.unlink(missing_ok=True)
            if n in present:
                p.write_text('#!/bin/sh\nexit 0\n')
                p.chmod(0o755)
        log = base / 'log'
        log.unlink(missing_ok=True)
        script = 'set -eu\nroot=' + shlex.quote(str(base)) + '''
fail(){ printf '%s\\n' "$1" >&2; exit 1; }
install_packages(){
 for name in "$@";do
  printf 'install %s\\n' "$name" >> "$TEST_LOG"
  printf '#!/bin/sh\\nexit 0\\n' > "$TEST_BIN/$name"
  /bin/chmod 755 "$TEST_BIN/$name"
 done
}
''' + fragment
        p = subprocess.run(['/bin/bash', '-c', script], env=dict(os.environ, PATH=str(binpath), TEST_LOG=str(log), TEST_BIN=str(binpath)), capture_output=True, text=True)
        assert p.returncode == 0, p.stderr
        lines = log.read_text().splitlines()
        assert lines == ['install ' + n for n in ['tor', 'tmux'] if n not in present] + ['config'], (present, lines)
print('PASS: source installer checks Tor and tmux, installs only missing dependencies, then invokes configuration helper (isolated fragment fixtures)')
