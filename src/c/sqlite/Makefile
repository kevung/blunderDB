mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(dir $(mkfile_path))
.PHONY: all clean
TARGET         ?= i686-w64-mingw32
TARGET_CC      ?= $(TARGET)-gcc
TARGET_CXX     ?= $(TARGET)-g++
TARGET_AR      ?= $(TARGET)-ar
TARGET_NM      ?= $(TARGET)-nm
TARGET_LD      ?= $(TARGET)-g++
TARGET_AR      ?= $(TARGET)-ar
TARGET_RANLIB  ?= $(TARGET)-ranlib
TARGET_STRIP   ?= $(TARGET)-strip
TARGET_WINDRES ?= $(TARGET)-windres

CFLAGS = -O3

SQLITE_TARGET = $(current_dir)sqlite3.o
SQLITE_SRC = $(current_dir)sqlite3.c
SQLITE_HEADER = $(current_dir)sqlite3.h
SQLITE_LIB = $(current_dir)libsqlite3.a
SQLITE_INCDIR = $(current_dir)
SQLITE_LIBDIR = $(current_dir)

all: $(SQLITE_LIB)

$(SQLITE_LIB): $(SQLITE_TARGET)
	$(TARGET_AR) rvs $(SQLITE_LIB) $(SQLITE_TARGET)

$(SQLITE_TARGET): $(SQLITE_SRC)
	$(TARGET_CC) $(CFLAGS) -c $(SQLITE_SRC) -o $(SQLITE_TARGET)


clean:
	rm -f $(SQLITE_TARGET)
