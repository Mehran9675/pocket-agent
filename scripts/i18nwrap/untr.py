# untr.py FILE:LINE... removes tr(...) around string literals on those lines.
import re, sys
for spec in sys.argv[1:]:
    path, line = spec.rsplit(':', 1)
    lines = open(path).read().split('\n')
    i = int(line) - 1
    lines[i] = re.sub(r'tr\(("(?:[^"\\]|\\.)*")\)', r'\1', lines[i])
    open(path, 'w').write('\n'.join(lines))
