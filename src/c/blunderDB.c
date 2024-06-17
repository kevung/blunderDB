#include <stdlib.h>
#include <iup.h>
#include <iupdraw.h>
#include <cd.h>
#include <cdiup.h>

cdCanvas *winCanvas = NULL;
cdCanvas *curCanvas = NULL;

/************************ Interface ***********************/

#define DEFAULT_SIZE "400x300"


static int canvas_action(Ihandle* ih)
{
    int i, w, h;
    cdCanvas *canvas;

    canvas = cdCreateCanvas(CD_IUP, ih);
    cdCanvasGetSize(canvas, &w, &h, NULL, NULL);

    cdCanvasBackground(canvas, CD_BLUE);
    cdCanvasClear(canvas);

    cdCanvasLineWidth(canvas, 3);
    cdCanvasLineStyle(canvas, CD_CONTINUOUS);
    cdCanvasForeground(canvas, cdEncodeAlpha(CD_DARK_MAGENTA, 128));
    cdCanvasRect(canvas, 100, 200, 100, 200);

    cdCanvasSetAttribute(canvas, "DRAWCOLOR", "252 186 3");

    cdCanvasFlush(canvas);

    return IUP_DEFAULT;
}


/************************ Main ****************************/
int main(int argc, char **argv)
{
  Ihandle *dlg, *hbox, *vbox, *label;
  Ihandle *canvas;

  IupOpen(&argc, &argv);

  canvas = IupCanvas(NULL);
  IupSetAttribute(canvas, "NAME", "CANVAS");
  IupSetAttribute(canvas, "EXPAND", "YES");

  vbox = IupVbox(canvas, NULL);
  IupSetAttribute(vbox, "NMARGIN", "10x10");
  IupSetAttribute(vbox, "GAP", "10");

  dlg = IupDialog(vbox);
  IupSetAttribute(dlg, "TITLE", "blunderDB");
  IupSetAttribute(dlg, "SIZE", DEFAULT_SIZE);

  /* Registers callbacks */
  IupSetCallback(canvas, "ACTION", (Icallback)canvas_action);


  IupShowXY(dlg, IUP_CENTER, IUP_CENTER);

  IupMainLoop();

  IupClose();
  return EXIT_SUCCESS;
}
