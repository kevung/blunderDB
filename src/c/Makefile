TARGET = i686-w64-mingw32
include ./iup/Makefile
include ./sqlite/Makefile

all:

SRC = blunderDB.c
EXE = blunderDB.exe

WINDOWS_LIBS = -lgdi32 -lcomdlg32 -lcomctl32 -luuid -loleaut32 -lole32

# $(info SQLITE_LIB is $(SQLITE_LIB))
# $(info SQLITE_LIBDIR is $(SQLITE_LIBDIR))
# $(info SQLITE_INCDIR is $(SQLITE_INCDIR))
# $(info LIBIUP is $(LIBIUP))


$(EXE): $(SRC) $(LIBIUP) $(SQLITE_LIB)
	$(TARGET_CC) -mwindows -static -I$(IUP_INCDIR) -I$(CD_INCDIR) -I$(FREETYPE_INCDIR) -I$(ZLIB_INCDIR) -I$(SQLITE_INCDIR) -o $(EXE) $(SRC) -L$(IUP_LIBDIR) -L$(CD_LIBDIR) -L$(FREETYPE_LIBDIR) -L$(ZLIB_LIBDIR) -L$(SQLITE_LIBDIR) -liupcd -lcd -lcdcontextplus -liup -lfreetype6 -lz -lsqlite3 $(WINDOWS_LIBS)
	$(TARGET_STRIP) $(EXE)

all: $(EXE)
