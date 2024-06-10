#include <stdlib.h>
#include <iup.h>
#include <iupdraw.h>
#include <cd.h>
#include <cdiup.h>

cdCanvas *winCanvas = NULL;
cdCanvas *curCanvas = NULL;

/************************ Interface ***********************/

#define DEFAULT_SIZE "400x300"


static int canvas_action(Ihandle *ih)
{
    int i, w, h;

    IupDrawBegin(ih);

    IupDrawGetSize(ih, &w, &h);

    IupSetAttribute(ih, "DRAWCOLOR", "252 186 3");
    IupSetAttribute(ih, "DRAWSTYLE", "FILL");
    IupDrawRectangle(ih, 0, 0, w-1, h-1);


    IupDrawEnd(ih);
    return IUP_DEFAULT;
}


/************************ Main ****************************/
int main(int argc, char **argv)
{
  Ihandle *dlg, *hbox, *vbox, *label;
  Ihandle *canvas;

  IupOpen(&argc, &argv);

  /* canvas = IupCanvas(NULL); */
  canvas = cdCreateCanvas(CD_IUP, NULL);
  IupSetAttribute(canvas, "NAME", "CANVAS");
  IupSetAttribute(canvas, "EXPAND", "YES");

  vbox = IupVbox(canvas, NULL);
  IupSetAttribute(vbox, "NMARGIN", "10x10");
  IupSetAttribute(vbox, "GAP", "10");

  dlg = IupDialog(vbox);
  IupSetAttribute(dlg, "TITLE", "BlunderDB");
  IupSetAttribute(dlg, "SIZE", DEFAULT_SIZE);

  /* Registers callbacks */
  IupSetCallback(canvas, "ACTION", (Icallback)canvas_action);


  IupShowXY(dlg, IUP_CENTER, IUP_CENTER);

  IupMainLoop();

  IupClose();
  return EXIT_SUCCESS;
}
