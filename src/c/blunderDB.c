#include <stdlib.h>
#include <iup.h>
#include <iupdraw.h>
#include <cd.h>
#include <cdiup.h>

cdCanvas *winCanvas = NULL;
cdCanvas *curCanvas = NULL;

/************************ Interface ***********************/

#define DEFAULT_SIZE "800x600"


static int canvas_action_cb(Ihandle* ih)
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

static int item_new_action_cb(void)
{
    IupMessage("New position!", "Bye");
    return IUP_DEFAULT;
}


/************************ Main ****************************/
int main(int argc, char **argv)
{
  Ihandle *dlg, *hbox, *vbox, *label;
  Ihandle *text;
  Ihandle *menu, *submenu_file, *submenu_edit,
          *submenu_position, *submenu_match,
          *submenu_search, *submenu_tool,
          *submenu_help;

  Ihandle *menu_file;
  Ihandle *item_new, *item_open, *item_recent, *item_close;
  Ihandle *item_import, *item_import_wizard;
  Ihandle *item_save, *item_saveas;
  Ihandle *item_export;
  Ihandle *item_properties;
  Ihandle *item_exit;

  Ihandle *menu_edit;
  Ihandle *item_undo, *item_redo, *item_copy, *item_cut, *item_paste;
  Ihandle *item_editmode;

  Ihandle *menu_position;
  Ihandle *item_next_position, *item_prev_position,
          *item_new_position,
          *item_import_position, *item_import_position_bybatch;
  Ihandle *item_new_collection;
  Ihandle *item_delete_collection;
  Ihandle *item_add_collection;

  Ihandle *menu_match;
  Ihandle *item_import_match, *item_import_match_bybatch, 
          *item_match_library;

  Ihandle *menu_search;
  Ihandle *item_search_blunder, *item_search_structure,
          *item_search_dice, *item_search_cube, *item_search_score,
          *item_search_player, *item_search_engine;
  Ihandle *item_searchmode;

  Ihandle *menu_tool;
  Ihandle *item_find_position_without_analysis;
  Ihandle *item_preferences;

  Ihandle *menu_help;
  Ihandle *item_manual, *item_userguide, *item_tips, *item_cmdmode;
  Ihandle *item_keyboard;
  Ihandle *item_getinvolved, *item_donate;
  Ihandle *item_about;

  Ihandle *canvas;

  IupOpen(&argc, &argv);

  text = IupText(NULL);
  IupSetAttributes(text, "VALUE = \"This text is here a sample\", EXPAND = YES");

  /* item_new = IupItem("New", NULL); */
  /* item_open = IupItem("Open", NULL); */
  /* item_recent = IupItem("Recent", NULL); */
  /* item_close = IupItem("Close", NULL); */
  /* item_import = IupItem("Import...", NULL); */
  /* item_import_wizard = IupItem("Import Wizard", NULL); */
  /* item_save = IupItem("Save", NULL); */
  /* item_saveas = IupItem("Save As...", NULL); */
  /* item_export = IupItem("Export...", NULL); */
  /* item_properties = IupItem("Properties", NULL); */
  /* item_exit = IupItem("Exit", "item_exit_act"); */
  /* menu_file = IupMenu(item_new, item_open, item_recent, item_close, */
  /*         IupSeparator(), item_save, item_saveas, */ 
  /*         IupSeparator(), item_import, item_import_wizard, */ 
  /*         IupSeparator(), item_export, */ 
  /*         IupSeparator(), item_properties, */
  /*         IupSeparator, item_exit, NULL); */

  /* item_undo = IupItem("Undo", NULL); */
  /* item_redo = IupItem("Redo", NULL); */
  /* item_copy = IupItem("Copy", NULL); */
  /* item_cut = IupItem("Cut", NULL); */
  /* item_paste = IupItem("Paste", NULL); */
  /* item_editmode = IupItem("Edit Mode", NULL); */
  /* menu_edit = IupMenu(item_undo, item_redo, */
  /*         item_copy, item_cut, item_paste, */
  /*         IupSeparator(), item_editmode, NULL); */

  /* item_next_position = IupItem("Next Position", NULL); */
  /* item_prev_position = IupItem("Previous Position", NULL); */
  /* item_new_position = IupItem("New Position", NULL); */
  /* item_import_position = IupItem("Import", NULL); */
  /* item_import_position_bybatch = IupItem("Import by Batch", NULL); */
  /* item_new_collection = IupItem("New Library", NULL); */
  /* item_delete_collection = IupItem("Delete Library", NULL); */
  /* item_add_collection = IupItem("Add to Library", NULL); */
  /* menu_position = IupMenu(item_next_position, item_prev_position, */ 
  /*         item_new_position, item_import_position, */
  /*         item_import_position_bybatch, IupSeparator(), */
  /*         item_new_collection, item_delete_collection, */
  /*         item_add_collection, NULL); */

  /* item_import_match = IupItem("Import Match", NULL); */
  /* item_import_match_bybatch = IupItem("Import Matches by Batch", */
  /*         NULL); */
  /* item_match_library = IupItem("Match Library", NULL); */
  /* menu_match = IupMenu(item_import_match, item_import_match_bybatch, */
  /*         item_match_library, NULL); */

  /* item_search_blunder = IupItem("by Blunder", NULL); */
  /* item_search_structure = IupItem("by Dice", NULL); */
  /* item_search_cube = IupItem("by Cube Decision", NULL); */
  /* item_search_score = IupItem("by Score", NULL); */
  /* item_search_player = IupItem("by Player", NULL); */
  /* item_search_engine = IupItem("Search Engine", NULL); */
  /* item_searchmode = IupItem("Search Mode", NULL); */
  /* menu_search = IupMenu(item_search_blunder, */
  /*         item_search_structure, item_search_cube, */
  /*         item_search_score, item_search_player, */
  /*         item_search_engine, IupSeparator(), */
  /*         item_searchmode, NULL); */

  /* item_find_position_without_analysis = IupItem("Find Positions without Analysis", NULL); */
  /* item_preferences = IupItem("Preferences", NULL); */
  /* menu_tool = IupMenu(item_find_position_without_analysis, */
  /*         IupSeparator(), item_preferences, NULL); */

  /* item_manual = IupItem("Help Manual", NULL); */
  /* item_userguide = IupItem("User Guide", NULL); */
  /* item_tips = IupItem("Tips", NULL); */
  /* item_cmdmode = IupItem("Command Mode Help", NULL); */
  /* item_keyboard = IupItem("Keyboard shortcuts", NULL); */
  /* item_getinvolved = IupItem("Get Involved", NULL); */
  /* item_donate = IupItem("Donate to blunderDB", NULL); */
  /* item_about = IupItem("About", NULL); */
  /* menu_help = IupMenu(item_manual, item_userguide, */
  /*         item_tips, item_cmdmode, item_keyboard, */
  /*         IupSeparator(), item_getinvolved, item_donate, */
  /*         IupSeparator(), item_about, NULL); */

  /* submenu_file = IupSubmenu("File", menu_file); */
  /* submenu_edit = IupSubmenu("Edit", menu_edit); */
  /* submenu_position = IupSubmenu("Positions", menu_position); */
  /* submenu_match = IupSubmenu("Matches", menu_match); */
  /* submenu_search = IupSubmenu("Search", menu_search); */
  /* submenu_help = IupSubmenu("Help", menu_help); */

  /* menu = IupMenu(submenu_file, submenu_edit, submenu_position, */
  /*         submenu_match, submenu_search, submenu_tool, submenu_help, */
  /*         NULL); */

  item_new = IupItem("New", NULL);
  menu_file = IupMenu(item_new, NULL);
  submenu_file = IupSubmenu("File", menu_file);
  menu = IupMenu(submenu_file, NULL);
  IupSetHandle("menu", menu);
  IupSetCallback(item_new, "ACTION", (Icallback) item_new_action_cb);

  canvas = IupCanvas(NULL);
  IupSetAttribute(canvas, "NAME", "CANVAS");
  IupSetAttribute(canvas, "EXPAND", "YES");

  vbox = IupVbox(canvas, NULL);
  IupSetAttribute(vbox, "NMARGIN", "10x10");
  IupSetAttribute(vbox, "GAP", "10");

  dlg = IupDialog(vbox);
  IupSetAttribute(dlg, "TITLE", "blunderDB");
  IupSetAttribute(dlg, "SIZE", DEFAULT_SIZE);
  IupSetAttribute(dlg, "MENU", "menu");

  /* Registers callbacks */
  IupSetCallback(canvas, "ACTION", (Icallback)canvas_action_cb);


  IupShowXY(dlg, IUP_CENTER, IUP_CENTER);

  IupMainLoop();

  IupClose();
  return EXIT_SUCCESS;
}
