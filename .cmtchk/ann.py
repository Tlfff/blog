import json,re,sys,io
# spec: { path: { "eol": {"12":"text",...}, "before": {"30":["l1","l2"],...} } }
# line numbers refer to the ORIGINAL file (1-based). Applied bottom-up so numbers stay valid.
def run(spec):
    known = {"eol", "before", "inner", "set"}
    for path, ops in spec.items():
        unknown = set(ops) - known
        if unknown:
            print("UNKNOWN-KEYS %s %s" % (path, sorted(unknown)))
            raise SystemExit(1)
        with io.open(path, encoding='utf-8') as f:
            lines = f.read().split('\n')
        eol = {int(k): v for k, v in ops.get("eol", {}).items()}
        setl = {int(k): v for k, v in ops.get("set", {}).items()}
        bef = {int(k): v for k, v in ops.get("before", {}).items()}
        for k, v in ops.get("inner", {}).items():
            bef[int(k)] = v
        bad = [n for n in list(eol) + list(bef) + list(ops.get("set", {}).keys() and [int(x) for x in ops.get("set", {})]) if n < 1 or n > len(lines)]
        if bad:
            print("BADLINE %s %s" % (path, bad)); continue
        for n, txt in sorted(setl.items()):
            lines[n-1] = txt
        for n, txt in sorted(eol.items()):
            base = lines[n-1].rstrip()
            base = re.sub(r'\s*//.*$', '', base)
            lines[n-1] = base + " // " + txt

        # guard: warn if a "before" insert lands where a comment already exists above
        for n in sorted(bef):
            prev = lines[n-2].strip() if n >= 2 else ''
            if prev.startswith('//'):
                print("WARN-DUP %s:%d prev=%s" % (path, n, prev[:60]))
        for n in sorted(bef, reverse=True):
            # 目标行为空行时，向下取第一个非空行的缩进，避免注释顶格
            src = n-1
            while src < len(lines) and lines[src].strip() == '':
                src += 1
            if src >= len(lines):
                src = n-1
            indent = re.match(r'[\t ]*', lines[src]).group(0)
            t = bef[n]
            t = t if isinstance(t, list) else [t]
            lines[n-1:n-1] = [(indent + "// " + x) if x else "" for x in t]
        with io.open(path, 'w', encoding='utf-8') as f:
            f.write('\n'.join(lines))
        print("OK %s (+%d eol, +%d block)" % (path, len(eol), len(bef)))
if __name__ == '__main__':
    run(json.load(open(sys.argv[1], encoding='utf-8')))
