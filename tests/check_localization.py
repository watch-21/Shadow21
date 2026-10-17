from pathlib import Path
import json,re
root=Path(__file__).resolve().parents[1]
a=json.loads((root/'gui/i18n/de.json').read_text());b=json.loads((root/'src/internal/i18n/de.json').read_text())
assert a==b,'CLI and UI dictionaries differ'
keys=set()
for p in (root/'gui/qml').rglob('*.qml'):
 if p.name.startswith('Modern') or p.name=='InfoCard.qml':continue
 for m in re.finditer(r'AppLanguage\.tr\(((?:"(?:\\.|[^"\\])*"\s*(?:\+\s*)?)+)\)',p.read_text()):
  key=''.join(json.loads(s) for s in re.findall(r'"(?:\\.|[^"\\])*"',m[1]))
  if key.strip() and key not in {'shadow21','SHADOW21','VERSION '}:keys.add(key)
assert len(keys)>40
assert all(k in a and a[k] for k in keys),sorted(keys-set(a))
for f in ['gui/org.shadow21.Desktop.desktop','gui/org.shadow21.Cli.desktop']:
 text=(root/f).read_text();assert 'Comment[de]=' in text and 'anonsurf' not in text.lower()
main=(root/'gui/qml/app/Main.qml').read_text()
for component in ['HeaderBar','ProtectionPanel','ConnectionOverview','ActionPanel','TmuxPanel','MacPanel']:assert component+' {' in main
assert 'StackLayout' not in main and 'Dashboard' not in main
panel=(root/'gui/qml/dashboard/ActionPanel.qml').read_text()
for label in ['New Identity','Verify IP','Restart','Refresh']:assert 'AppLanguage.tr("'+label+'")' in panel
assert panel.count('ActionButton {')==4
assert 'Tor Monitor' not in panel
assert 'Tor Monitor' not in (root/'gui/src/tray/traymenu.cpp').read_text()
mac=(root/'gui/qml/dashboard/MacPanel.qml').read_text()
assert 'AppLanguage.tr("Check MAC")' not in mac
assert 'root.workspace.currentMAC || AppLanguage.tr("Unavailable")' in mac
print(f'PASS: four desktop action buttons and main panels retained, {len(keys)} localized QML strings, {len(a)} shared translations')
