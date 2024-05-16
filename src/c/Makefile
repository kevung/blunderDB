APPS =				\
	blunderDB

CFLAGS  = -g -Wall

CFLAGS  += `pkg-config gtk+-3.0 --cflags`
LDFLAGS += `pkg-config gtk+-3.0 --libs`

all: $(APPS)

clean:
	$(RM): $(APPS)
