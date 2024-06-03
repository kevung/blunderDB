TARGET = i686-w64-mingw32
include ./iup/Makefile

all:

SRC = hello-world.c
EXE = hello-world.exe

WINDOWS_LIBS = -lgdi32 -lcomdlg32 -lcomctl32 -luuid -loleaut32 -lole32

$(EXE): $(SRC) $(LIBIUP)
	$(TARGET_CC) -mwindows -static -I$(IUP_INCDIR) -o $(EXE) $(SRC) -L$(IUP_LIBDIR) -liup $(WINDOWS_LIBS)
	$(TARGET_STRIP) $(EXE)

all: $(EXE)
