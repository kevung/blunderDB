# IUP Build System

[IUP](http://webserver2.tecgraf.puc-rio.br/iup/) is a pretty cool cross-platform
GUI toolkit, it's become by go-to for cross-compiling small windows programs.

Its build system is fairly different, so I wrote this - it's a Makefile that
pulls down the IUP, CD, and IM sources, compiles them, and (optionally) copies
the libraries and headers into a single folder.

Right now, this only supports cross-compiling to a Windows target (ie, with
`i686-w64-mingw32-gcc`. I'm still working on Linux support using the OpenMotif toolkit - I'm having
a hard time getting OpenMotif to cross-compile, patches are welcome!

## Requirements

* curl
* GNU make
* tar
* patch
* a cross-compiler

## Usage

If you want, you can just type `make install` - this will download
IUP, CD, IM, zlib, freetype, and ftgl, compile them, and create a folder
named `output/$TARGET/` with a `lib` and `include` folder, with all the
libraries and headers, respectively.

You can also include this Makefile in your own makefile:

```make
TARGET = i686-w64-mingw32
include iup/Makefile

# iup/Makefile will set a bunch of variables
# based on your TARGET, like TARGET_CC, TARGET_AR, etc

# the iup makefile sets an 'all' target, be sure
# to override it

all:

SRC = hello-world.c
EXE = hello-world.exe

WINDOWS_LIBS = -lgdi32 -lcomdlg32 -lcomctl32 -luuid -loleaut32 -lole32

$(EXE): $(SRC) $(LIBIUP)
	$(TARGET_CC) -static -I$(IUP_INCDIR) -o $(EXE) $(SRC) -L$(IUP_LIBDIR) -liup $(WINDOWS_LIBS)
	$(TARGET_STRIP) $(EXE)

all: $(EXE)
```

## LICENSE

This makefile released under an MIT license (see `LICENSE`).
