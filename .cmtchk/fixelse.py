import re,io,sys,subprocess
# 把落在 "} else {" / "} else if" / "case ...:" 之前的编号注释，下移到块内首行
files = subprocess.run(["git","diff","--name-only","HEAD","--","*.go"],capture_output=True,text=True).stdout.split()
tot=0
for p in files:
    if p.startswith('.cmtchk/'): continue
    try:
        lines=io.open(p,encoding='utf-8').read().split('\n')
    except Exception: continue
    out=[];moved=0;i=0
    while i < len(lines):
        cur=lines[i].strip()
        nxt=lines[i+1].strip() if i+1<len(lines) else ''
        if re.match(r'^// \d+(\.\d+)? ', cur) and (nxt.startswith('} else') or nxt.startswith('}else')):
            # 注释后紧跟 else：把注释移到 else 块内部第一行
            indent=re.match(r'[\t ]*',lines[i]).group(0)
            out.append(lines[i+1])
            out.append(indent+'\t'+cur)
            moved+=1;i+=2;continue
        out.append(lines[i]);i+=1
    if moved:
        io.open(p,'w',encoding='utf-8').write('\n'.join(out))
        print("FIXED",p,"moved=%d"%moved);tot+=moved
print("total-moved",tot)
