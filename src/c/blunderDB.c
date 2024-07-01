#include <stdbool.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <iup.h>
#include <iupdraw.h>
#include <cd.h>
#include <cdiup.h>
#include <sqlite3.h>

char db_file[10240];
cdCanvas *winCanvas = NULL;
cdCanvas *curCanvas = NULL;

/************************ Database ***********************/
sqlite3 *db = NULL;
bool is_db_saved = true;
int rc;
char *errMsg = 0;

const char *sql_library =
"CREATE TABLE library ("
"id INTEGER PRIMARY KEY AUTOINCREMENT,"
"name TEXT,"
"position_list_id INTEGR,"
"FOREIGN KEY(position_list_id) REFERENCES position_list(id)"
");";

const char *sql_position_list =
"CREATE TABLE position_list ("
"id INTEGER PRIMARY KEY AUTOINCREMENT,"
"position_id INTEGER,"
"FOREIGN KEY(position_id) REFERENCES position(id)"
");";

const char *sql_player = 
"CREATE TABLE player ("
"id INTEGER PRIMARY KEY AUTOINCREMENT,"
"name TEXT"
");";


const char *sql_position =
"CREATE TABLE position ("
"id INTEGER PRIMARY KEY AUTOINCREMENT,"
"p0 INTEGER,"
"p1 INTEGER,"
"p2 INTEGER,"
"p3 INTEGER,"
"p4 INTEGER,"
"p5 INTEGER,"
"p6 INTEGER,"
"p7 INTEGER,"
"p8 INTEGER,"
"p9 INTEGER,"
"p10 INTEGER,"
"p11 INTEGER,"
"p12 INTEGER,"
"p13 INTEGER,"
"p14 INTEGER,"
"p15 INTEGER,"
"p16 INTEGER,"
"p17 INTEGER,"
"p18 INTEGER,"
"p19 INTEGER,"
"p20 INTEGER,"
"p21 INTEGER,"
"p22 INTEGER,"
"p23 INTEGER,"
"p24 INTEGER,"
"p25 INTEGER,"
"player1_id INTEGER,"
"player2_id INTEGER,"
"player1_score INTEGER,"
"player2_score INTEGER,"
"cube_position INTEGER,"
"comment TEXT,"
"FOREIGN KEY(player1_id) REFERENCES player(id),"
"FOREIGN KEY(player2_id) REFERENCES player(id)"
");";

;

void execute_sql(sqlite3 *db, const char *sql)
{
    rc = sqlite3_exec(db, sql, 0, 0, &errMsg);
    if(rc != SQLITE_OK) {
        printf("SQL error: %s\n", errMsg);
    } else {
        printf("SQL executed successfully\n");
    }
}

int db_create(const char* filename)
{
    if (remove(filename) == 0) {
        printf("Existing database file removed successfully\n");
    } else {
        printf("No existing database file to remove, or failed to remove\n");
    }

    rc = sqlite3_open(filename, &db);
    printf("%s\n", sql_position);

    if(rc) {
        printf("Can't create database: %s\n", sqlite3_errmsg(db));
        return rc;
    } else {
        printf("Created database successfully\n");
    }

    printf("Try to create player table.\n");
    execute_sql(db, sql_player);

    printf("Try to create position table.\n");
    execute_sql(db, sql_position);

    printf("Try to create position_list table.\n");
    execute_sql(db, sql_position_list);

    printf("Try to create library table.\n");
    execute_sql(db, sql_library);

    return 0;
}

int db_open(const char* filename)
{
    rc = sqlite3_open(filename, &db);

    if(rc) {
        printf("Can't open database: %s\n", sqlite3_errmsg(db));
        return rc;
    } else {
        printf("Opened database successfully\n");
    }

    return 0;

}

int db_close(sqlite3 *db)
{
    rc = sqlite3_close(db);
    if (rc != SQLITE_OK) {
        printf("Can't close database. Maybe already closed. Err: %s\n", sqlite3_errmsg(db));
    } else {
        printf("Closed database successfully\n");
    }

}

/************************ Interface ***********************/

#define DEFAULT_SIZE "800x600"

Ihandle *dlg, *hbox, *vbox, *label;
Ihandle *text;
Ihandle *menu, *submenu_file, *submenu_edit,
        *submenu_position, *submenu_match,
        *submenu_search, *submenu_tool,
        *submenu_help;

Ihandle *menu_file;
Ihandle *item_new, *item_open, *item_recent, *item_exit;
Ihandle *item_import;
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
Ihandle *item_new_library;
Ihandle *item_delete_library;
Ihandle *item_add_library;

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

Ihandle *toolbar_hb;
Ihandle *btn_new, *btn_open, *btn_save, *btn_close, *btn_properties;
Ihandle *btn_cut, *btn_copy, *btn_paste;
Ihandle *btn_undo, *btn_redo;
Ihandle *btn_prev, *btn_next;
Ihandle *btn_blunder, *btn_dice, *btn_cube, *btn_score, *btn_player;
Ihandle *btn_preferences;
Ihandle *btn_manual;

Ihandle *canvas;

Ihandle *lbl_statusbar;

static int canvas_action_cb(Ihandle* ih);
static int item_new_action_cb(void);
static int item_open_action_cb(void);
static int item_recent_action_cb(void);
static int item_save_action_cb(void);
static int item_saveas_action_cb(void);
static int item_import_action_cb(void);
static int item_export_action_cb(void);
static int item_properties_action_cb(void);
static int item_exit_action_cb();
static int item_undo_action_cb();
static int item_redo_action_cb();
static int item_copy_action_cb();
static int item_cut_action_cb();
static int item_paste_action_cb();
static int item_editmode_action_cb();
static int item_nextposition_action_cb();
static int item_prevposition_action_cb();
static int item_newposition_action_cb();
static int item_importposition_action_cb();
static int item_importpositionbybatch_action_cb();
static int item_newlibrary_action_cb();
static int item_deletelibrary_action_cb();
static int item_addtolibrary_action_cb();
static int item_importmatch_action_cb();
static int item_importmatchbybatch_action_cb();
static int item_matchlibrary_action_cb();
static int item_searchblunder_action_cb();
static int item_searchdice_action_cb();
static int item_searchcubedecision_action_cb();
static int item_searchscore_action_cb();
static int item_searchplayer_action_cb();
static int item_searchengine_action_cb();
static int item_searchmode_action_cb();
static int item_findpositionwithoutanalysis_action_cb();
static int item_preferences_action_cb();
static int item_helpmanual_action_cb();
static int item_userguide_action_cb();
static int item_tips_action_cb();
static int item_commandmodehelp_action_cb();
static int item_keyboardshortcuts_action_cb();
static int item_getinvolved_action_cb();
static int item_donatetoblunderdb_action_cb();
static int item_about_action_cb();
void error_callback(void);



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

    Ihandle *filedlg;

    filedlg = IupFileDlg();
    IupSetAttribute(filedlg, "DIALOGTYPE", "SAVE");
    IupSetAttribute(filedlg, "TITLE", "New Database");
    IupSetAttribute(filedlg, "EXTFILTER",
            "Blunder Database (.db)|*.db|All Files|*.*|");
    IupSetAttribute(filedlg, "EXTDEFAULT", ".db");
    IupPopup(filedlg, IUP_CENTER, IUP_CENTER);
    
    switch(IupGetInt(filedlg, "STATUS"))
    {
        case 1: // new file
        case 0 : // file already exists
            const char *db_filename = IupGetAttribute(filedlg, "VALUE");
            int result = db_create(db_filename);
            if (result != 0) {
                printf("Database creation failed\n");
                return result;
            }
            printf("Database created successfully\n");
            break; 

        case -1 : 
            printf("IupFileDlg","Operation Canceled");
            return 1;
            break; 
    }

    IupDestroy(filedlg);
    return IUP_DEFAULT;
}


static int item_open_action_cb(void)
{
    Ihandle *filedlg;

    filedlg = IupFileDlg();
    IupSetAttribute(filedlg, "DIALOGTYPE", "OPEN");
    IupSetAttribute(filedlg, "TITLE", "Open Database");
    IupSetAttribute(filedlg, "EXTFILTER",
            "Blunder Database (.db)|*.db|All Files|*.*|");
    IupSetAttribute(filedlg, "EXTDEFAULT", ".db");
    IupPopup(filedlg, IUP_CENTER, IUP_CENTER);
    
    switch(IupGetInt(filedlg, "STATUS"))
    {
        case 1: // new file
            printf("Database does not exist.");
            break;
        case 0 : // file already exists
            const char *db_filename = IupGetAttribute(filedlg, "VALUE");
            int result = db_open(db_filename);
            if (result != 0) {
                printf("Database opening failed\n");
                return result;
            }
            printf("Database opened successfully\n");
            break; 

        case -1 : 
            printf("IupFileDlg","Operation Canceled");
            return 1;
            break; 
    }

    IupDestroy(filedlg);
    return IUP_DEFAULT;

}

static int item_recent_action_cb(void)
{
    error_callback();
}

static int item_import_action_cb(void)
{
    error_callback();
}

static int item_export_action_cb(void)
{
    error_callback();
}

static int item_properties_action_cb(void)
{
    error_callback();
}


static int item_save_action_cb(void)
{
    error_callback();
}

static int item_saveas_action_cb(void)
{
    error_callback();
}

static int item_exit_action_cb()
{
    // verify if db is saved with is_db_saved before quitting.

    db_close(db);
    IupClose();
    return EXIT_SUCCESS;
}

static int item_undo_action_cb(void)
{
    error_callback();
}

static int item_redo_action_cb(void)
{
    error_callback();
}

static int item_copy_action_cb(void)
{
    error_callback();
}

static int item_cut_action_cb(void)
{
    error_callback();
}

static int item_paste_action_cb(void)
{
    error_callback();
}

static int item_editmode_action_cb(void)
{
    error_callback();
}

static int item_nextposition_action_cb(void)
{
    error_callback();
}

static int item_prevposition_action_cb(void)
{
    error_callback();
}

static int item_newposition_action_cb(void)
{
    error_callback();
}

static int item_importposition_action_cb(void)
{
    error_callback();
}

static int item_importpositionbybatch_action_cb(void)
{
    error_callback();
}

static int item_newlibrary_action_cb(void)
{
    error_callback();
}

static int item_deletelibrary_action_cb(void)
{
    error_callback();
}

static int item_addtolibrary_action_cb(void)
{
    error_callback();
}

static int item_importmatch_action_cb(void)
{
    error_callback();
}

static int item_importmatchbybatch_action_cb(void)
{
    error_callback();
}

static int item_matchlibrary_action_cb(void)
{
    error_callback();
}

static int item_searchblunder_action_cb(void)
{
    error_callback();
}

static int item_searchdice_action_cb(void)
{
    error_callback();
}

static int item_searchcubedecision_action_cb(void)
{
    error_callback();
}

static int item_searchscore_action_cb(void)
{
    error_callback();
}

static int item_searchplayer_action_cb(void)
{
    error_callback();
}

static int item_searchengine_action_cb(void)
{
    error_callback();
}

static int item_searchmode_action_cb(void)
{
    error_callback();
}

static int item_findpositionwithoutanalysis_action_cb(void)
{
    error_callback();
}

static int item_preferences_action_cb(void)
{
    error_callback();
}

static int item_helpmanual_action_cb(void)
{
    error_callback();
}

static int item_userguide_action_cb(void)
{
    error_callback();
}

static int item_tips_action_cb(void)
{
    error_callback();
}

static int item_commandmodehelp_action_cb(void)
{
    error_callback();
}

static int item_keyboardshortcuts_action_cb(void)
{
    error_callback();
}

static int item_getinvolved_action_cb(void)
{
    error_callback();
}

static int item_donate_action_cb(void)
{
    error_callback();
}

static int item_about_action_cb(void)
{
    error_callback();
}


void error_callback(void)
{
    IupMessage("Callback Error", "Functionality not implemented yet!");
}

/************************ Main ****************************/
int main(int argc, char **argv)
{
  IupOpen(&argc, &argv);
  IupImageLibOpen();
  IupSetLanguage("ENGLISH");

  /* Define menus */
  item_new = IupItem("&New Database\tCtrl+N", NULL);
  item_open = IupItem("&Open Database\tCtrl+O", NULL);
  item_recent = IupItem("Recent D&atabase", NULL);
  item_save = IupItem("&Save Database", NULL);
  item_saveas = IupItem("Save &As...", NULL);
  item_import = IupItem("&Import...", NULL);
  item_export = IupItem("&Export...", NULL);
  item_properties = IupItem("Database &Metadata...", NULL);
  item_exit = IupItem("E&xit\tCtrl+Q", NULL);
  menu_file = IupMenu(item_new, item_open, item_recent,
          IupSeparator(), item_import,
          IupSeparator(), item_export,
          IupSeparator(), item_save, item_saveas,
          IupSeparator(), item_properties,
          IupSeparator(), item_exit, NULL);
  submenu_file = IupSubmenu("&File", menu_file);

  item_undo = IupItem("&Undo\tCtrl-Z", NULL);
  item_redo = IupItem("&Redo\tCtrl-Y", NULL);
  item_copy = IupItem("Co&py\tCtrl-C", NULL);
  item_cut = IupItem("Cu&t\tCtrl-X", NULL);
  item_paste = IupItem("Pa&ste\tCtrl-V", NULL);
  item_editmode = IupItem("&Edit Mode\tCtrl-E", NULL);
  menu_edit = IupMenu(item_undo, item_redo,
          item_copy, item_cut, item_paste,
          IupSeparator(), item_editmode, NULL);
  submenu_edit = IupSubmenu("&Edit", menu_edit);

  item_next_position = IupItem("Ne&xt Position", NULL);
  item_prev_position = IupItem("Pre&vious Position", NULL);
  item_new_position = IupItem("Ne&w Position", NULL);
  item_import_position = IupItem("&Import Position", NULL);
  item_import_position_bybatch = IupItem("Import Positions by &Batch", NULL);
  item_new_library = IupItem("New &Library", NULL);
  item_delete_library = IupItem("&Delete Library", NULL);
  item_add_library = IupItem("&Add to Library", NULL);
  menu_position = IupMenu(item_next_position, item_prev_position, 
          item_new_position, IupSeparator(), item_import_position, 
          item_import_position_bybatch, IupSeparator(),
          item_new_library, item_delete_library,
          item_add_library, NULL);
  submenu_position = IupSubmenu("&Positions", menu_position);

  item_import_match = IupItem("&Import Match", NULL);
  item_import_match_bybatch = IupItem("Import Matches by &Batch",
          NULL);
  item_match_library = IupItem("Match &Library", NULL);
  menu_match = IupMenu(item_import_match, item_import_match_bybatch,
          item_match_library, NULL);
  submenu_match = IupSubmenu("&Matches", menu_match);

  item_search_blunder = IupItem("by &Blunder", NULL);
  item_search_dice = IupItem("by &Dice", NULL);
  item_search_cube = IupItem("by &Cube Decision", NULL);
  item_search_score = IupItem("by &Score", NULL);
  item_search_player = IupItem("by &Player", NULL);
  item_search_engine = IupItem("Search &Engine", NULL);
  item_searchmode = IupItem("Search &Mode", NULL);
  menu_search = IupMenu(item_search_blunder,
          item_search_dice, item_search_cube,
          item_search_score, item_search_player,
          item_search_engine, IupSeparator(),
          item_searchmode, NULL);
  submenu_search = IupSubmenu("&Search", menu_search);

  item_find_position_without_analysis = IupItem("&Find Positions without Analysis", NULL);
  item_preferences = IupItem("&Preferences", NULL);
  menu_tool = IupMenu(item_find_position_without_analysis,
          IupSeparator(), item_preferences, NULL);
  submenu_tool = IupSubmenu("&Tools", menu_tool);

  item_manual = IupItem("Help &Manual", NULL);
  item_userguide = IupItem("&User Guide", NULL);
  item_tips = IupItem("&Tips", NULL);
  item_cmdmode = IupItem("&Command Mode Help", NULL);
  item_keyboard = IupItem("&Keyboard shortcuts", NULL);
  item_getinvolved = IupItem("Get &Involved", NULL);
  item_donate = IupItem("&Donate to blunderDB", NULL);
  item_about = IupItem("&About", NULL);
  menu_help = IupMenu(item_manual, item_userguide,
          item_tips, item_cmdmode, item_keyboard,
          IupSeparator(), item_getinvolved, item_donate,
          IupSeparator(), item_about, NULL);
  submenu_help = IupSubmenu("&Help", menu_help);

  menu = IupMenu(submenu_file, submenu_edit, submenu_position,
          submenu_match, submenu_search, submenu_tool, submenu_help,
          NULL);

  IupSetHandle("menu", menu);

  /* Define toolbar */

  btn_new = IupButton(NULL, NULL);
  IupSetAttribute(btn_new, "IMAGE", "IUP_FileNew");
  IupSetAttribute(btn_new, "FLAT", "Yes");
  IupSetAttribute(btn_new, "CANFOCUS", "No");
  IupSetAttribute(btn_new, "TIP", "New Database");
  btn_open = IupButton(NULL, NULL);
  IupSetAttribute(btn_open, "IMAGE", "IUP_FileOpen");
  IupSetAttribute(btn_open, "FLAT", "Yes");
  IupSetAttribute(btn_open, "CANFOCUS", "No");
  IupSetAttribute(btn_open, "TIP", "Open Database");
  btn_save = IupButton(NULL, NULL);
  IupSetAttribute(btn_save, "IMAGE", "IUP_FileSave");
  IupSetAttribute(btn_save, "FLAT", "Yes");
  IupSetAttribute(btn_save, "CANFOCUS", "No");
  IupSetAttribute(btn_save, "TIP", "Save Database");
  btn_close = IupButton(NULL, NULL);
  IupSetAttribute(btn_close, "IMAGE", "IUP_FileClose");
  IupSetAttribute(btn_close, "FLAT", "Yes");
  IupSetAttribute(btn_close, "CANFOCUS", "No");
  IupSetAttribute(btn_close, "TIP", "Close Database");
  btn_properties = IupButton(NULL, NULL);
  IupSetAttribute(btn_properties, "IMAGE", "IUP_FileProperties");
  IupSetAttribute(btn_properties, "FLAT", "Yes");
  IupSetAttribute(btn_properties, "CANFOCUS", "No");
  IupSetAttribute(btn_properties, "TIP", "Database Metadata");
  btn_cut = IupButton(NULL, NULL);
  IupSetAttribute(btn_cut, "IMAGE", "IUP_EditCut");
  IupSetAttribute(btn_cut, "FLAT", "Yes");
  IupSetAttribute(btn_cut, "CANFOCUS", "No");
  IupSetAttribute(btn_cut, "TIP", "Cut Position");
  btn_copy = IupButton(NULL, NULL);
  IupSetAttribute(btn_copy, "IMAGE", "IUP_EditCopy");
  IupSetAttribute(btn_copy, "FLAT", "Yes");
  IupSetAttribute(btn_copy, "CANFOCUS", "No");
  IupSetAttribute(btn_copy, "TIP", "Copy Position");
  btn_paste = IupButton(NULL, NULL);
  IupSetAttribute(btn_paste, "IMAGE", "IUP_EditPaste");
  IupSetAttribute(btn_paste, "FLAT", "Yes");
  IupSetAttribute(btn_paste, "CANFOCUS", "No");
  IupSetAttribute(btn_paste, "TIP", "Paste Position");
  btn_undo = IupButton(NULL, NULL);
  IupSetAttribute(btn_undo, "IMAGE", "IUP_EditUndo");
  IupSetAttribute(btn_undo, "FLAT", "Yes");
  IupSetAttribute(btn_undo, "CANFOCUS", "No");
  IupSetAttribute(btn_undo, "TIP", "Undo");
  btn_redo = IupButton(NULL, NULL);
  IupSetAttribute(btn_redo, "IMAGE", "IUP_EditRedo");
  IupSetAttribute(btn_redo, "FLAT", "Yes");
  IupSetAttribute(btn_redo, "CANFOCUS", "No");
  IupSetAttribute(btn_redo, "TIP", "Redo");
  btn_prev = IupButton(NULL, NULL);
  IupSetAttribute(btn_prev, "IMAGE", "IUP_ArrowLeft");
  IupSetAttribute(btn_prev, "FLAT", "Yes");
  IupSetAttribute(btn_prev, "CANFOCUS", "No");
  IupSetAttribute(btn_prev, "TIP", "Previous Position");
  btn_next = IupButton(NULL, NULL);
  IupSetAttribute(btn_next, "IMAGE", "IUP_ArrowRight");
  IupSetAttribute(btn_next, "FLAT", "Yes");
  IupSetAttribute(btn_next, "CANFOCUS", "No");
  IupSetAttribute(btn_next, "TIP", "Next Position");
  btn_blunder = IupButton("Blunder", NULL);
  IupSetAttribute(btn_blunder, "FLAT", "Yes");
  IupSetAttribute(btn_blunder, "CANFOCUS", "No");
  IupSetAttribute(btn_blunder, "TIP", "Search by Blunder");
  btn_dice = IupButton("Dice", NULL);
  IupSetAttribute(btn_dice, "FLAT", "Yes");
  IupSetAttribute(btn_dice, "CANFOCUS", "No");
  IupSetAttribute(btn_dice, "TIP", "Search by Dice");
  btn_cube = IupButton("Cube", NULL);
  IupSetAttribute(btn_cube, "FLAT", "Yes");
  IupSetAttribute(btn_cube, "CANFOCUS", "No");
  IupSetAttribute(btn_cube, "TIP", "Search by Cube");
  btn_score = IupButton("Score", NULL);
  IupSetAttribute(btn_score, "FLAT", "Yes");
  IupSetAttribute(btn_score, "CANFOCUS", "No");
  IupSetAttribute(btn_score, "TIP", "Search by Score");
  btn_player = IupButton("Player", NULL);
  IupSetAttribute(btn_player, "FLAT", "Yes");
  IupSetAttribute(btn_player, "CANFOCUS", "No");
  IupSetAttribute(btn_player, "TIP", "Search by Player");
  btn_preferences = IupButton(NULL, NULL);
  IupSetAttribute(btn_preferences, "IMAGE", "IUP_ToolsSettings");
  IupSetAttribute(btn_preferences, "FLAT", "Yes");
  IupSetAttribute(btn_preferences, "CANFOCUS", "No");
  IupSetAttribute(btn_preferences, "TIP", "Preferences");
  btn_manual = IupButton(NULL, NULL);
  IupSetAttribute(btn_manual, "IMAGE", "IUP_MessageHelp");
  IupSetAttribute(btn_manual, "FLAT", "Yes");
  IupSetAttribute(btn_manual, "CANFOCUS", "No");
  IupSetAttribute(btn_manual, "TIP", "Help Manual");
  toolbar_hb = IupHbox(
          btn_new, btn_open, btn_save, btn_close, btn_properties,
          IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
          btn_cut, btn_copy, btn_paste,
          IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
          btn_undo, btn_redo,
          IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
          btn_prev, btn_next,
          IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
          btn_blunder, btn_dice, btn_cube, btn_score, btn_player,
          IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
          btn_preferences,
          IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
          btn_manual,
          NULL);
  IupSetAttribute(toolbar_hb, "NAME", "TOOLBAR");
  IupSetAttribute(toolbar_hb, "MARGIN", "5x5");
  IupSetAttribute(toolbar_hb, "GAP", "2");


  /* Define main canvas */
  canvas = IupCanvas(NULL);
  IupSetAttribute(canvas, "NAME", "CANVAS");
  IupSetAttribute(canvas, "EXPAND", "YES");

  /* Define status bar */
  lbl_statusbar = IupLabel("NORMAL MODE");
  IupSetAttribute(lbl_statusbar, "NAME", "STATUSBAR");
  IupSetAttribute(lbl_statusbar, "EXPAND", "HORIZONTAL");
  IupSetAttribute(lbl_statusbar, "PADDIND", "10x5");

  /* General layout */
  vbox = IupVbox(toolbar_hb, canvas, lbl_statusbar, NULL);
  IupSetAttribute(vbox, "NMARGIN", "10x10");
  IupSetAttribute(vbox, "GAP", "10");

  /* Main Windows */
  dlg = IupDialog(vbox);
  IupSetAttribute(dlg, "TITLE", "blunderDB");
  IupSetAttribute(dlg, "SIZE", DEFAULT_SIZE);
  IupSetAttribute(dlg, "MENU", "menu");

  /* Registers callbacks */
  IupSetCallback(dlg, "K_cN", (Icallback) item_new_action_cb);
  IupSetCallback(dlg, "K_cO", (Icallback) item_open_action_cb);
  IupSetCallback(dlg, "K_cS", (Icallback) item_save_action_cb);
  IupSetCallback(dlg, "K_cQ", (Icallback) item_exit_action_cb);
  IupSetCallback(dlg, "K_cZ", (Icallback) item_undo_action_cb);
  IupSetCallback(item_new, "ACTION", (Icallback) item_new_action_cb);
  IupSetCallback(btn_new, "ACTION", (Icallback) item_new_action_cb);
  IupSetCallback(item_open, "ACTION", (Icallback) item_open_action_cb);
  IupSetCallback(btn_open, "ACTION", (Icallback) item_open_action_cb);
  IupSetCallback(item_recent, "ACTION", (Icallback) item_recent_action_cb);
  IupSetCallback(item_import, "ACTION", (Icallback) item_import_action_cb);
  IupSetCallback(item_export, "ACTION", (Icallback) item_export_action_cb);
  IupSetCallback(item_save, "ACTION", (Icallback) item_save_action_cb);
  IupSetCallback(btn_save, "ACTION", (Icallback) item_save_action_cb);
  IupSetCallback(item_saveas, "ACTION", (Icallback) item_saveas_action_cb);
  IupSetCallback(item_properties, "ACTION", (Icallback) item_properties_action_cb);
  IupSetCallback(btn_properties, "ACTION", (Icallback) item_properties_action_cb);
  IupSetCallback(item_exit, "ACTION", (Icallback) item_exit_action_cb);
  IupSetCallback(btn_close, "ACTION", (Icallback) item_exit_action_cb);
  IupSetCallback(item_undo, "ACTION", (Icallback) item_undo_action_cb);
  IupSetCallback(btn_undo, "ACTION", (Icallback) item_undo_action_cb);
  IupSetCallback(item_redo, "ACTION", (Icallback) item_redo_action_cb);
  IupSetCallback(btn_redo, "ACTION", (Icallback) item_redo_action_cb);
  IupSetCallback(item_cut, "ACTION", (Icallback) item_cut_action_cb);
  IupSetCallback(btn_cut, "ACTION", (Icallback) item_cut_action_cb);
  IupSetCallback(item_copy, "ACTION", (Icallback) item_copy_action_cb);
  IupSetCallback(btn_copy, "ACTION", (Icallback) item_copy_action_cb);
  IupSetCallback(item_paste, "ACTION", (Icallback) item_paste_action_cb);
  IupSetCallback(btn_paste, "ACTION", (Icallback) item_paste_action_cb);
  IupSetCallback(item_editmode, "ACTION", (Icallback) item_editmode_action_cb);
  IupSetCallback(item_next_position, "ACTION", (Icallback) item_nextposition_action_cb);
  IupSetCallback(btn_next, "ACTION", (Icallback) item_nextposition_action_cb);
  IupSetCallback(item_prev_position, "ACTION", (Icallback) item_prevposition_action_cb);
  IupSetCallback(btn_prev, "ACTION", (Icallback) item_prevposition_action_cb);
  IupSetCallback(item_new_position, "ACTION", (Icallback) item_newposition_action_cb);
  IupSetCallback(item_import_position, "ACTION", (Icallback) item_importposition_action_cb);
  IupSetCallback(item_import_position_bybatch, "ACTION", (Icallback) item_importpositionbybatch_action_cb);
  IupSetCallback(item_new_library, "ACTION", (Icallback) item_newlibrary_action_cb);
  IupSetCallback(item_delete_library, "ACTION", (Icallback) item_deletelibrary_action_cb);
  IupSetCallback(item_add_library, "ACTION", (Icallback) item_addtolibrary_action_cb);
  IupSetCallback(item_import_match, "ACTION", (Icallback) item_importmatch_action_cb);
  IupSetCallback(item_import_match_bybatch, "ACTION", (Icallback) item_importmatchbybatch_action_cb);
  IupSetCallback(item_match_library, "ACTION", (Icallback) item_matchlibrary_action_cb);
  IupSetCallback(item_search_blunder, "ACTION", (Icallback) item_searchblunder_action_cb);
  IupSetCallback(btn_blunder, "ACTION", (Icallback) item_searchblunder_action_cb);
  IupSetCallback(item_search_dice, "ACTION", (Icallback) item_searchdice_action_cb);
  IupSetCallback(btn_dice, "ACTION", (Icallback) item_searchdice_action_cb);
  IupSetCallback(item_search_cube, "ACTION", (Icallback) item_searchcubedecision_action_cb);
  IupSetCallback(btn_cube, "ACTION", (Icallback) item_searchcubedecision_action_cb);
  IupSetCallback(item_search_score, "ACTION", (Icallback) item_searchscore_action_cb);
  IupSetCallback(btn_score, "ACTION", (Icallback) item_searchscore_action_cb);
  IupSetCallback(item_search_player, "ACTION", (Icallback) item_searchplayer_action_cb);
  IupSetCallback(btn_player, "ACTION", (Icallback) item_searchplayer_action_cb);
  IupSetCallback(item_search_engine, "ACTION", (Icallback) item_searchengine_action_cb);
  IupSetCallback(item_searchmode, "ACTION", (Icallback) item_searchmode_action_cb);
  IupSetCallback(item_find_position_without_analysis, "ACTION", (Icallback) item_findpositionwithoutanalysis_action_cb);
  IupSetCallback(item_preferences, "ACTION", (Icallback) item_preferences_action_cb);
  IupSetCallback(btn_preferences, "ACTION", (Icallback) item_preferences_action_cb);
  IupSetCallback(item_manual, "ACTION", (Icallback) item_helpmanual_action_cb);
  IupSetCallback(btn_manual, "ACTION", (Icallback) item_helpmanual_action_cb);
  IupSetCallback(item_userguide, "ACTION", (Icallback) item_userguide_action_cb);
  IupSetCallback(item_tips, "ACTION", (Icallback) item_tips_action_cb);
  IupSetCallback(item_cmdmode, "ACTION", (Icallback) item_commandmodehelp_action_cb);
  IupSetCallback(item_keyboard, "ACTION", (Icallback) item_keyboardshortcuts_action_cb);
  IupSetCallback(item_getinvolved, "ACTION", (Icallback) item_getinvolved_action_cb);
  IupSetCallback(item_donate, "ACTION", (Icallback) item_donate_action_cb);
  IupSetCallback(item_about, "ACTION", (Icallback) item_about_action_cb);
  IupSetCallback(canvas, "ACTION", (Icallback)canvas_action_cb);


  IupShowXY(dlg, IUP_CENTER, IUP_CENTER);

  IupMainLoop();

  db_close(db);
  IupClose();
  return EXIT_SUCCESS;
}
