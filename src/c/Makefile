TARGET = i686-w64-mingw32
include ./iup/Makefile

all:

SRC = blunderDB.c
EXE = blunderDB.exe

WINDOWS_LIBS = -lgdi32 -lcomdlg32 -lcomctl32 -luuid -loleaut32 -lole32

$(EXE): $(SRC) $(LIBIUP)
	$(TARGET_CC) -mwindows -static -I$(IUP_INCDIR) -I$(CD_INCDIR) -I$(FREETYPE_INCDIR) -I$(ZLIB_INCDIR) -o $(EXE) $(SRC) -L$(IUP_LIBDIR) -L$(CD_LIBDIR) -L$(FREETYPE_LIBDIR) -L$(ZLIB_LIBDIR) -liup -lcd -liupcd -lcdcontextplus -lfreetype6 -lz $(WINDOWS_LIBS)
	$(TARGET_STRIP) $(EXE)

all: $(EXE)
