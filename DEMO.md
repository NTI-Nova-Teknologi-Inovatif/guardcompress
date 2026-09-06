# Demo GuardCompress (Windows PowerShell)

```powershell
$env:GUARDCOMPRESS_BIN="C:\tools\GuardCompress\core\bin\guardcompress-windows-amd64.exe"
# 1. Cek sistem
& $env:GUARDCOMPRESS_BIN doctor
# 2. File bersih (PNG 208 byte) -> harusnya clean exit 0
& $env:GUARDCOMPRESS_BIN check --in C:\Users\allmi\AppData\Local\Temp\opencode\clean.png --out-dir ./tmp/demo-clean --json
# 3. File kotor (EICAR) -> harusnya blocked exit 2
& $env:GUARDCOMPRESS_BIN check --in C:\Users\allmi\AppData\Local\Temp\opencode\eicar.txt --out-dir ./tmp/demo-block --json
# 4. Wrapper Node
node -e "const gc=require('./wrappers/node/src/index.js'); console.log(gc.process('C:/Users/allmi/AppData/Local/Temp/opencode/clean.png',{}).report.status)"
# 5. Wrapper Python
$env:PYTHONPATH="wrappers/python/src"; python -c "from guardcompress import process; print(process(r'C:\Users\allmi\AppData\Local\Temp\opencode\clean.png',{})['report']['status'])"
# 6. Contoh integrasi per bahasa
& $env:GUARDCOMPRESS_BIN init --lang php
```
