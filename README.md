This tool allows packing VDF/MOD files for the Gothic Game series (Gothic 1 & 2)

## Usage

Help output
```
> vdfsbuilder.exe -h
example:
vdfsbuilder.exe [options] *.vm

options:
  -b string
        base directory (substitution for ".\")
  -o string
        override output filepath
  -ts string
        a Timestamp in the format "YYYY-MM-dd HH:mm:ss". E.g "2021-11-28 12:31:40"
```

Given the following `Scripts.vm` file, a call to this tool might look like this:

Scripts.vm
```ini
[BEGINVDF]
Comment=This is a comment for the VDF that will be generated
BaseDir=.\
VDFName=.\Scripts.vdf
[FILES]
# Try to include everything from _WORK\*
_Work\*
*.md -r
[EXCLUDE]
DESKTOP.INI -r
# exclude all *.md & *.txt files
*.md
*.txt
[INCLUDE]
# after excluding, allow README.md if it's in BaseDir
README.md
# also allow all ocurrences of "notes.txt" in every subdirectory or BaseDir
notes.txt -r
[ENDVDF]
```

Commandline call:
```cmd
                   overriden "BaseDir"
                   ||                       || custom output filename
                   \/                       \/                   \/ custom timestamp     \/ Path to the vm file
> vdfsbuilder.exe -b "C:\modding\gothic\" -o "Scripts v44.vdf" -ts "2033-12-31 23:56:33" Scripts.vm
```

## Usage in Github Actions

See here for a full example with versioning and publishing a release:  
https://github.com/kirides/ninja-manareg/blob/master/.github/workflows/release-linux.yml

```yaml
name: Build Vdf

# on:
# ...

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - uses: kirides/vdfsbuilder@f6402aa7c633f4f657bc67e22efe2f0e3caa6802
        with:
          in: example.vm
          # out: custom_name.vdf # optional
          # baseDir: src # optional
          # ts: '2037-01-01 12:00:00' # optional

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: dist
          path: example.vdf # name of the vdf file
```
