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
_Work\* -r
[EXCLUDE]
DESKTOP.INI -r
[INCLUDE]
[ENDVDF]
```

Commandline call:
```cmd
                   overriden "BaseDir"
                   ||                       || custom output filename
                   \/                       \/                   \/ custom timestamp     \/ Path to the vm file
> vdfsbuilder.exe -b "C:\modding\gothic\" -o "Scripts v44.vdf" -ts "2033-12-31 23:56:33" Scripts.vm
```
