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

      - uses: kirides/vdfsbuilder@f7438d978d2974f9c1e97fd6dd3419709795c9d2
        with:
          in: example.vm
          # out: custom_name.vdf # optional
          # baseDir: src # optional

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: dist
          path: example.vdf # name of the vdf file
```
