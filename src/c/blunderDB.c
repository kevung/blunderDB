#include <ctype.h>
#include <stdbool.h>
#include <math.h>
#include <stdbool.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <iup.h>
#include <iupdraw.h>
#include <iupcontrols.h>
#include <cd.h>
#include <cdiup.h>
#include <wd.h>
#include <sqlite3.h>

/************************** Data *************************/

#define PLAYER1 1
#define PLAYER2 -1
#define PLAYER1_POINTLABEL "*abcdefghijklmnopqrstuvwxyz"
#define PLAYER2_POINTLABEL "YABCDEFGHIJKLMNOPQRSTUVWX*Z"

typedef struct
{
    int checker[26];
    int cube;
    int p1_score; // 2=2-away; 1=crawford; 0=postcrawford; -1=unlimited;
    int p2_score;
    int dice[2];
    int is_double;
    int is_take;
    int is_on_roll;
} POSITION;

const POSITION POS_DEFAULT = {
    .checker = {0,
        -2, 0, 0, 0, 0, 5,
        0, 3, 0, 0, 0, -5,
        5, 0, 0, 0, -3, 0,
        -5, 0, 0, 0, 0, 2,
        0},
    .cube = 0,
    .p1_score = -1,
    .p2_score = -1,
    .dice = {0, 0},
    .is_double = 0,
    .is_take = 0,
    .is_on_roll = 0,
};

const POSITION POS_VOID = {
    .checker = {0,
        0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0,
        0},
    .cube = 0,
    .p1_score = -1,
    .p2_score = -1,
    .dice = {0, 0},
    .is_double = 0,
    .is_take = 0,
    .is_on_roll = 0,
};

POSITION pos;
POSITION *pos_ptr;
bool is_pointletter_active = false;

int char_in_string(const char c, const char* s)
{
    int index;
    char *e;
    e = strchr(s, c);
    index = (int) (e - s);
    return index;
}

void pos_print(const POSITION* p)
{
    printf("checker:\n");
    for(int i=0; i<26; i++)
    {
        printf("%i: %i\n", i, p->checker[i]);
    }
    printf("cube: %i\n", p->cube);
    printf("p1_score: %i\n", p->p1_score);
    printf("p2_score: %i\n", p->p2_score);
    printf("dice: %i, %i\n", p->dice[0], p->dice[1]);
    printf("is_double: %i\n", p->is_double);
    printf("is_take: %i\n", p->is_take);
    printf("is_on_roll: %i\n", p->is_on_roll);
}

void int_swap(int* i, int* j)
{
    int t;
    t = *i;
    *i = *j;
    *j = t;
}

// BUG? Check if it work for crawford, post-crawford, unlimited scores
char* pos_to_str(const POSITION* p)
{
    const char p1[27] = PLAYER1_POINTLABEL;
    const char p2[27] = PLAYER2_POINTLABEL;
    char p1_score[2];
    char p2_score[2];
    char _d[2];
    char* c = malloc(100 * sizeof(char));
    memcpy(c, "\0", 1);
    sprintf(p1_score, "%d", p->p1_score);
    sprintf(p2_score, "%d", p->p2_score);
    snprintf(c, sizeof(c), "%s,%s", p1_score, p2_score);
    strcat(c, ":");
    int a;
    for(int i=26; i>=0; i--)
    {
        a = p->checker[i];
        if(i==26) a = p->cube;
        if(a>0) {
            _d[0] = p1[i];
            _d[1] = '\0';
            strcat(c, _d);
            sprintf(_d, "%d", a);
            strcat(c, _d);
        } else if (a<0) {
            _d[0] = p2[i];
            _d[1] = '\0';
            strcat(c, _d);
            sprintf(_d, "%d", -a);
            strcat(c, _d);
        }
    }
    return c;
}

// BUG? Check if it work for crawford, post-crawford, unlimited scores
char* pos_to_str_paren(const POSITION* p)
{
    const char p1[27] = PLAYER1_POINTLABEL;
    const char p2[27] = PLAYER2_POINTLABEL;
    char p1_score[2];
    char p2_score[2];
    char _d[2];
    char* c = malloc(100 * sizeof(char));
    char* c_spare = malloc(50 * sizeof(char));
    char* c_point = malloc(50 * sizeof(char));
    memcpy(c, "\0", 1);
    memcpy(c_spare, "\0", 1);
    memcpy(c_point, "\0", 1);
    sprintf(p1_score, "%d", p->p1_score);
    sprintf(p2_score, "%d", p->p2_score);
    snprintf(c, sizeof(c), "%s,%s", p1_score, p2_score);
    strcat(c, ":");
    int a;

    /* put into string checkers and points */
    void f(int a, char* spare, char* point, char *d)
    {
        switch (a)
        {
            case 1:
                strcat(spare, d);
                sprintf(d, "%d", a);
                strcat(spare, d);
                break;
            case 2:
                strcat(point, d);
                break;
            default:
                strcat(point, d);
                strcat(spare, d);
                sprintf(d, "%d", a-2);
                strcat(spare, d);
                break;
        }
    }

    for(int i=26; i>=0; i--)
    {
        a = p->checker[i];
        if(i==26) a = p->cube;
        if(a>0) {
            _d[0] = p1[i];
            _d[1] = '\0';
            f(a, c_spare, c_point, _d);
        } else if (a<0) {
            _d[0] = p2[i];
            _d[1] = '\0';
            f(-a, c_spare, c_point, _d);
        }
    }
    strcat(c, "(");
    strcat(c, c_point);
    strcat(c, ")");
    strcat(c, c_spare);
    free(c_point);
    free(c_spare);
    return c;
}


// BUG? Check if it work for crawford, post-crawford, unlimited scores
int str_to_pos(const char* s, POSITION* pos)
{
    const char p1[27] = PLAYER1_POINTLABEL;
    const char p2[27] = PLAYER2_POINTLABEL;
    // i_score index symbol ":". If none, -1 so i_score+1=0.
    int has_score = 0, i_score = -1;
    char s_p1_score[5], s_p2_score[5];
    s_p1_score[0] = '\0';
    s_p2_score[0] = '\0';
    int i, j = 0;
    int len = strlen(s);
    *pos = POS_VOID;
    /* detect score */
    for(int i=0; i<len; i++)
    {
        if(s[i]==':') {
            has_score = 1;
            i_score = i;
            break;
        }
    }
    if(has_score)
    {
        j=0;
        /* find , to delimit scores */
        for(int i=0; i<i_score; i++)
        {
            if(s[i]==',') {
                j=i;
                break;
            }
        }
        for(int i=0; i<j; i++)
        {
            if(!isdigit(s[i]) && s[i]!='-') return 0; //fail
            s_p1_score[i] = s[i];
            s_p1_score[i+1] = '\0';
        }
        for(int i=j+1; i<i_score; i++)
        {
            if(!isdigit(s[i]) && s[i]!='-') return 0; //fail
            s_p2_score[i-j-1] = s[i];
            s_p2_score[i-j] = '\0';
        }
        pos->p1_score = atoi(s_p1_score);
        pos->p2_score = atoi(s_p2_score);
    }
    /* detect checkers */
    int paren_open = 0;
    int hyphen_index = -1;
    int i_point = -1; // point to fill with checkers
    for(int i=i_score+1; i<len; i++) {
        if(!isalnum(s[i])) {
            if(s[i]=='(') { paren_open = 1; }
            else if(s[i]==')') { paren_open = 0; }
            else if(s[i]=='-') {
                if(isalpha(s[i-1]) && isalpha(s[i+1])
                        && ((islower(s[i-1]) && islower(s[i+1]))
                            || (isupper(s[i-1]) && isupper(s[i+1])))) 
                { hyphen_index = i; }
                else { return 0; } //error
            } else { return 0; } //error
        }
        else if (isalpha(s[i])) {
            i_point = char_in_string(tolower(s[i]), p1);
            if(s[i]=='Y') i_point = 0; //p2 bar
            if(paren_open==1) {
                if(islower(s[i])) pos->checker[i_point] += 2;  
                if(isupper(s[i])) pos->checker[i_point] -= 2;  
                if(hyphen_index > -1) {
                    int upper_point = i_point;
                    int lower_point = char_in_string(tolower(s[i-2]), p1);
                    if(upper_point<lower_point) int_swap(&upper_point, &lower_point);
                    for(int k=lower_point+1; k<upper_point; k++) {
                        if(islower(s[i])) pos->checker[k] += 2;  
                        if(isupper(s[i])) pos->checker[k] -= 2;  
                    }
                    hyphen_index = -1; //reset
                }
                i_point = -1; //reset
            } else if (paren_open==0) {
                if(s[i+1]!='\0' && (!isdigit(s[i+1]))) {
                    if(s[i]!='z' && s[i]!='Z') {
                        if(islower(s[i])) pos->checker[i_point] += 1;  
                        if(isupper(s[i])) pos->checker[i_point] -= 1;  
                        if(hyphen_index > -1) {
                            int upper_point = i_point;
                            int lower_point = char_in_string(tolower(s[i-2]), p1);
                            if(upper_point<lower_point) int_swap(&upper_point, &lower_point);
                            for(int k=lower_point+1; k<upper_point; k++) {
                                if(islower(s[i])) pos->checker[k] += 1;  
                                if(isupper(s[i])) pos->checker[k] -= 1;  
                            }
                            hyphen_index = -1; //reset
                        }
                        i_point = -1; //reset
                    } else {
                        if(islower(s[i])) pos->cube += 1;  
                        if(isupper(s[i])) pos->cube -= 1;  
                        i_point = -1; //reset
                    }
                }
            } else { return 0; } //error
        } else if (isdigit(s[i])) {
            if(paren_open==1) { return 0; }
            else if(paren_open==0) {
                if(isalpha(s[i-1]) && !(isdigit(s[i+1]))) {
                    if(s[i]!='z' && s[i]!='Z') {
                        if(islower(s[i-1])) pos->checker[i_point] += (s[i] -'0');  
                        if(isupper(s[i-1])) pos->checker[i_point] -= (s[i] -'0');  
                    } else {
                        if(islower(s[i])) pos->cube += (s[i] -'0');  
                        if(isupper(s[i])) pos->cube -= (s[i] -'0');  
                    }
                }
                if(isalpha(s[i-1]) && isdigit(s[i+1])) {
                    if(s[i]!='z' && s[i]!='Z') {
                        if(islower(s[i-1])) pos->checker[i_point] += 10*(s[i]-'0')+(s[i+1]-'0');  
                        if(isupper(s[i-1])) pos->checker[i_point] -= 10*(s[i]-'0')+(s[i+1]-'0');  
                    } else {
                        if(islower(s[i])) pos->cube += (s[i] -'0');  
                        if(isupper(s[i])) pos->cube -= (s[i] -'0');  
                    }
                }
            } else { return 0; }
        } else { return 0; }
    }
    return 1; //success
}

void compute_pipcount(POSITION* pos, int* pip1, int* pip2){
    *pip1 = 0; *pip2 = 0;
    for(int i=0; i<26; i++){
        if(pos->checker[i]>0) {
            *pip1 += i*(pos->checker[i]);
        } else {
            *pip2 += (25-i)*abs(pos->checker[i]);
        }
    }
}

void compute_checkeroff(POSITION* pos, int* off1, int* off2){
    *off1 = 15; *off2 = 15;
    for(int i=0; i<26; i++){
        if(pos->checker[i]>0) *off1 -= abs(pos->checker[i]);
        if(pos->checker[i]<0) *off2 -= abs(pos->checker[i]);
    }
}

/************************ Database ***********************/

sqlite3 *db = NULL;
bool is_db_saved = true;
int rc;
char *errMsg = 0;
char db_file[10240];

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
    return rc;
}

/************************ Drawing *************************/

#define BOARD_WIDTH 13.
#define BOARD_HEIGHT 11.
#define BOARD_WITH_DECORATIONS_HEIGHT (BOARD_HEIGHT+2.*POINT_SIZE)
#define BOARD_DIRECTION 1
#define POINT_SIZE (BOARD_WIDTH/13.)
#define CHECKER_SIZE (0.95*POINT_SIZE)
#define CHECKER_LINECOLOR CD_BLACK
#define CHECKER_LINEWIDTH 3
#define CHECKER1_COLOR CD_BLACK
#define CHECKER2_COLOR CD_WHITE
#define CHECKER1_TEXTCOLOR CD_WHITE
#define CHECKER2_TEXTCOLOR CD_BLACK
#define CHECKER_TEXTLINEWIDTH 3
#define CHECKER_FONT "Times"
#define CHECKER_FONTSIZE 20
#define CHECKER_STYLE CD_PLAIN
#define BAR_WIDTH POINT_SIZE
#define BOARD_COLOR CD_BLACK
#define BOARD_LINEWIDTH 6
#define TRIANGLE_LINEWIDTH 2
#define TRIANGLE_LINECOLOR CD_BLACK
#define TRIANGLE1_COLOR CD_WHITE
#define TRIANGLE2_COLOR 0xd0d0d0
#define TRIANGLE2_HATCH 0
#define CUBE_LINEWIDTH 5
#define CUBE_TEXTLINEWIDTH 3
#define CUBE_LINECOLOR CD_BLACK
#define CUBE_SIZE (1.1*POINT_SIZE)
#define CUBE_FONT "Times"
#define CUBE_FONTSIZE 30
#define CUBE_STYLE CD_PLAIN
#define CUBE_XPOS (-BOARD_WIDTH/2. -1.5*POINT_SIZE)
#define CUBE_YPOS_CENTER 0.
#define CUBE_YPOS_UP (BOARD_HEIGHT/2. - CUBE_SIZE)
#define CUBE_YPOS_DOWN (-BOARD_HEIGHT/2.)
#define POINTNUMBER_FONT "Times"
#define POINTNUMBER_FONTSIZE 20
#define POINTNUMBER_STYLE CD_PLAIN
#define POINTNUMBER_LINECOLOR CD_BLACK
#define POINTNUMBER_YPOS_UP (BOARD_HEIGHT/2.+POINT_SIZE/2.)
#define POINTNUMBER_YPOS_DOWN -(POINTNUMBER_YPOS_UP)
#define SCORE_XPOS (BOARD_WIDTH/2.+1.7*POINT_SIZE)
#define SCORE_YPOS_UP (BOARD_HEIGHT/2.+0.1*POINT_SIZE)
#define SCORE_YPOS_DOWN (-SCORE_YPOS_UP)
#define SCORE_FONT "Times"
#define SCORE_FONTSIZE 20
/* #define SCORE_STYLE CD_PLAIN */
/* #define SCORE_STYLE CD_ITALIC */
#define SCORE_STYLE CD_BOLD
#define SCORE_LINECOLOR CD_BLACK
#define CHECKEROFF_XPOS (BOARD_WIDTH/2.+POINT_SIZE)
#define CHECKEROFF_YPOS_UP (3.*POINT_SIZE)
#define CHECKEROFF_YPOS_DOWN (-CHECKEROFF_YPOS_UP)
#define CHECKEROFF_FONT "Times"
#define CHECKEROFF_FONTSIZE 15
/* #define CHECKEROFF_STYLE CD_PLAIN */
#define CHECKEROFF_STYLE CD_ITALIC
#define CHECKEROFF_LINECOLOR CD_BLACK
#define PIPCOUNT_XPOS (-BOARD_WIDTH/2.-2.0*POINT_SIZE)
#define PIPCOUNT_YPOS_UP POINTNUMBER_YPOS_UP
#define PIPCOUNT_YPOS_DOWN (-PIPCOUNT_YPOS_UP)
#define PIPCOUNT_FONT "Times"
#define PIPCOUNT_FONTSIZE 15
/* #define PIPCOUNT_STYLE CD_PLAIN */
#define PIPCOUNT_STYLE CD_PLAIN
#define PIPCOUNT_LINECOLOR CD_BLACK

cdCanvas *cdv = NULL;
cdCanvas *db_cdv = NULL;

typedef struct {
    int button;
    int pressed;
    int x;
    int y;
    char* status;
} MOUSE;

void mouse_print(const MOUSE m){
    printf("mouse_print()\n");
    printf("button: %i\n", m.button);
    printf("pressed: %i\n", m.pressed);
    printf("x: %i\n", m.x);
    printf("y: %i\n", m.y);
    printf("status: %s\n", m.status);
}

MOUSE mouse;

void draw_triangle(cdCanvas *cv, const double x, const double y, const double up, const int long color){
    cdCanvasForeground(cv, color);
    cdCanvasBegin(cv, CD_FILL);
    wdCanvasVertex(cv, x, y);
    wdCanvasVertex(cv, x+POINT_SIZE, y);
    wdCanvasVertex(cv, x+POINT_SIZE/2, y + ((double) up)*5*POINT_SIZE);
    cdCanvasEnd(cv);

    cdCanvasLineWidth(cv, TRIANGLE_LINEWIDTH);
    cdCanvasForeground(cv, TRIANGLE_LINECOLOR);
    cdCanvasLineStyle(cv, CD_CONTINUOUS);
    cdCanvasBegin(cv, CD_CLOSED_LINES);
    wdCanvasVertex(cv, x, y);
    wdCanvasVertex(cv, x+POINT_SIZE, y);
    wdCanvasVertex(cv, x+POINT_SIZE/2, y + ((double) up)*5*POINT_SIZE);
    cdCanvasEnd(cv);
}

char* cubeText(const int value) {
    switch(value) {
        case 0:
            return "1";
        case 1:
            return "2";
        case 2:
            return "4";
        case 3:
            return "8";
        case 4:
            return "16";
        case 5:
            return "32";
        case 6:
            return "64";
        case 7:
            return "128";
        case 8:
            return "256";
        case 9:
            return "512";
        case 10:
            return "1024";
        default:
            return "?";
    }
}

void draw_cube(cdCanvas *cv, const int value){
    char* text = cubeText(abs(value));
    double x = CUBE_XPOS;
    double y = CUBE_YPOS_CENTER;
    if(value>0) y = CUBE_YPOS_DOWN;
    if(value<0) y = CUBE_YPOS_UP;
    cdCanvasForeground(cv, CUBE_LINECOLOR);
    cdCanvasLineStyle(cv, CD_CONTINUOUS);
    cdCanvasLineWidth(cv, CUBE_LINEWIDTH);
    cdCanvasLineJoin(cv, CD_ROUND);
    wdCanvasRect(cv, x, x+CUBE_SIZE, y, y+CUBE_SIZE);
    cdCanvasLineWidth(cv, CUBE_TEXTLINEWIDTH);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, CUBE_FONT, CUBE_STYLE, CUBE_FONTSIZE);
    wdCanvasText(cv, x+CUBE_SIZE/2, y+CUBE_SIZE/2, text);
}

void draw_board(cdCanvas* cv) {
    for(int i=0; i<3; i++){
        double x = -BOARD_WIDTH/2 +((double) i)*2*POINT_SIZE;
        double y = -BOARD_HEIGHT/2;
        cdCanvasForeground(cv, TRIANGLE1_COLOR);
        draw_triangle(cv, x, y, 1, TRIANGLE1_COLOR);
        draw_triangle(cv, x+POINT_SIZE, -y, -1, TRIANGLE1_COLOR);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2, y, 1, TRIANGLE1_COLOR);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2+POINT_SIZE, -y, -1,
                TRIANGLE1_COLOR);
    }

    if(TRIANGLE2_HATCH) cdCanvasHatch(cv, CD_HORIZONTAL);
    for(int i=0; i<3; i++){
        double x = -BOARD_WIDTH/2 +((double) i)*2*POINT_SIZE;
        double y = -BOARD_HEIGHT/2 +BOARD_HEIGHT;
        cdCanvasForeground(cv, TRIANGLE2_COLOR);
        draw_triangle(cv, x, y, -1, TRIANGLE2_COLOR);
        draw_triangle(cv, x+POINT_SIZE, -y, 1, TRIANGLE2_COLOR);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2, y, -1, TRIANGLE2_COLOR);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2+POINT_SIZE, -y, 1,
                TRIANGLE2_COLOR);
    }
    if(TRIANGLE2_HATCH) cdCanvasInteriorStyle(cv, CD_SOLID);

    cdCanvasForeground(cv, BOARD_COLOR);
    cdCanvasLineWidth(cv, BOARD_LINEWIDTH);
    cdCanvasLineStyle(cv, CD_CONTINUOUS);
    wdCanvasRect(cv, -BOARD_WIDTH/2, BOARD_WIDTH/2,
            -BOARD_HEIGHT/2, BOARD_HEIGHT/2);
    cdCanvasLineWidth(cv, BOARD_LINEWIDTH);
    wdCanvasRect(cv, -BAR_WIDTH/2, BAR_WIDTH/2,
            -BOARD_HEIGHT/2, BOARD_HEIGHT/2);
}

void draw_pointnumber(cdCanvas* cv, const int orientation) {
    double x, y;
    char t[3];
    cdCanvasForeground(cv, POINTNUMBER_LINECOLOR);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, POINTNUMBER_FONT, POINTNUMBER_STYLE, POINTNUMBER_FONTSIZE);
    if(orientation>0) {

        x = BOARD_WIDTH/2 -POINT_SIZE/2;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=1; i<7; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -POINT_SIZE;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=7; i<13; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -BOARD_WIDTH/2 +POINT_SIZE/2;
        y = POINTNUMBER_YPOS_UP;
        for(int i=13; i<19; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = POINT_SIZE;
        y = POINTNUMBER_YPOS_UP;
        for(int i=19; i<25; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

    } else {

        x = -BOARD_WIDTH/2 +POINT_SIZE/2;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=1; i<7; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = POINT_SIZE;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=7; i<13; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = BOARD_WIDTH/2 -POINT_SIZE/2;
        y = POINTNUMBER_YPOS_UP;
        for(int i=13; i<19; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -POINT_SIZE;
        y = POINTNUMBER_YPOS_UP;
        for(int i=19; i<25; i++){
            sprintf(t, "%d", i);
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }
    }
}

void draw_pointletter(cdCanvas* cv, const int orientation, const int cubevalue) {
    const char p1[27] = PLAYER1_POINTLABEL;
    double x, y;
    char t[2];
    cdCanvasForeground(cv, POINTNUMBER_LINECOLOR);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, POINTNUMBER_FONT, POINTNUMBER_STYLE, POINTNUMBER_FONTSIZE);
    t[1] = '\0';

    wdCanvasText(cv, 0, 0, "y");

    if(cubevalue>0) {
        wdCanvasText(cv, CUBE_XPOS -.5*POINT_SIZE, CUBE_YPOS_DOWN+CUBE_SIZE/2, "z");
    } else if(cubevalue<0) {
        wdCanvasText(cv, CUBE_XPOS -.5*POINT_SIZE, CUBE_YPOS_UP+CUBE_SIZE/2, "z");
    } else {
        wdCanvasText(cv, CUBE_XPOS -.5*POINT_SIZE, CUBE_YPOS_CENTER+CUBE_SIZE/2, "z");
    }

    if(orientation>0) {

        x = BOARD_WIDTH/2 -POINT_SIZE/2;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=1; i<7; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -POINT_SIZE;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=7; i<13; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -BOARD_WIDTH/2 +POINT_SIZE/2;
        y = POINTNUMBER_YPOS_UP;
        for(int i=13; i<19; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = POINT_SIZE;
        y = POINTNUMBER_YPOS_UP;
        for(int i=19; i<25; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

    } else {

        x = -BOARD_WIDTH/2 +POINT_SIZE/2;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=1; i<7; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = POINT_SIZE;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=7; i<13; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = BOARD_WIDTH/2 -POINT_SIZE/2;
        y = POINTNUMBER_YPOS_UP;
        for(int i=13; i<19; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -POINT_SIZE;
        y = POINTNUMBER_YPOS_UP;
        for(int i=19; i<25; i++){
            t[0] = p1[i];
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }
    }
}

void draw_score(cdCanvas* cv, const int score, const int player){
    char t[20];
    cdCanvasForeground(cv, SCORE_LINECOLOR);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, SCORE_FONT, SCORE_STYLE, SCORE_FONTSIZE);
    if(score>=2) {
        sprintf(t, "%d", score);
        strcat(t, " away");
    } else if(score==1) {
        t[0] = '\0';
        strcat(t, "\ncrawford");
    } else if(score==0) {
        t[0] = '\0';
        strcat(t, "\npost\ncrawford");
    } else {
        t[0] = '\0';
        strcat(t, "unlimited");
    }
    if(player>0) {
        wdCanvasText(cv, SCORE_XPOS, SCORE_YPOS_DOWN, t);
    } else {
        wdCanvasText(cv, SCORE_XPOS, SCORE_YPOS_UP, t);
    }
}

void draw_checkeroff(cdCanvas* cv, const int nb_off, const int player, const int orientation){
    char t[20], t2[10];
    cdCanvasForeground(cv, CHECKEROFF_LINECOLOR);
    if(nb_off<0) cdCanvasForeground(cv, CD_RED);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, CHECKEROFF_FONT, CHECKEROFF_STYLE, CHECKEROFF_FONTSIZE);
    t[0] = '('; t[1] = '\0';
    sprintf(t2, "%d", nb_off);
    strcat(t, t2);
    strcat(t, " OFF)");
    double x = CHECKEROFF_XPOS;
    if(orientation<1) x = -x;
    if(player>0) {
        wdCanvasText(cv, x, CHECKEROFF_YPOS_DOWN, t);
    } else {
        wdCanvasText(cv, x, CHECKEROFF_YPOS_UP, t);
    }
}

void draw_pipcount(cdCanvas* cv, const int pip, const int player){
    char t[10], t2[5];
    cdCanvasForeground(cv, PIPCOUNT_LINECOLOR);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, PIPCOUNT_FONT, PIPCOUNT_STYLE, PIPCOUNT_FONTSIZE);
    t[0] = '\0';
    strcat(t, "pip: ");
    sprintf(t2, "%d", pip);
    strcat(t, t2);
    if(player>0) {
        wdCanvasText(cv, PIPCOUNT_XPOS, PIPCOUNT_YPOS_DOWN, t);
    } else {
        wdCanvasText(cv, PIPCOUNT_XPOS, PIPCOUNT_YPOS_UP, t);
    }
}

/* ATTENTION TRAITER LE CAS SI PLUS DE 6 CHECKERS */
void draw_checker(cdCanvas* cv, const POSITION* p, const int dir) {
    double xc, yc, eps;

    if(BOARD_DIRECTION==1) eps = 1;
    if(BOARD_DIRECTION!=1) eps = -1;

    cdCanvasForeground(cv, CHECKER_LINECOLOR);
    cdCanvasLineWidth(cv, CHECKER_LINEWIDTH);
    cdCanvasLineStyle(cv, CD_CONTINUOUS);

    void draw_number_checkers(const double x, const double y, const int i) {
        char text[3];
        text[0]='\0';
        sprintf(text, "%d", i);
        cdCanvasLineWidth(cv, CHECKER_TEXTLINEWIDTH);
        cdCanvasTextAlignment(cv, CD_CENTER);
        cdCanvasFont(cv, CHECKER_FONT, CHECKER_STYLE, CHECKER_FONTSIZE);
        wdCanvasText(cv, x, y, text);
    }

    void draw_checker_samepoint(const double xc, const double yc,
            const int point, const double dir) {
        double _yc = yc; int q;
        int n=abs(p->checker[point]);
        if(n<=5) q=n;
        if(n>5) q=5;
        for(int k=0; k<q; k++) {
            if(p->checker[point]>0) {
                cdCanvasForeground(cv, CHECKER1_COLOR);
            } else {
                cdCanvasForeground(cv, CHECKER2_COLOR);
            }
            wdCanvasSector(cv, xc, _yc, CHECKER_SIZE, CHECKER_SIZE, 0, 360);
            cdCanvasForeground(cv, CHECKER_LINECOLOR);
            cdCanvasLineWidth(cv, CHECKER_LINEWIDTH);
            cdCanvasLineStyle(cv, CD_CONTINUOUS);
            wdCanvasArc(cv, xc, _yc, CHECKER_SIZE, CHECKER_SIZE, 0, 360);
            _yc += dir*CHECKER_SIZE;
        }
        if(n>5) {
            if(p->checker[point]>0) {
                cdCanvasForeground(cv, CHECKER1_TEXTCOLOR);
            } else {
                cdCanvasForeground(cv, CHECKER2_TEXTCOLOR);
            }
            draw_number_checkers(xc, yc+dir*4.*CHECKER_SIZE, n);
        }
    }

    void draw_checker_onbar(const int player) {
        int i, color; double dir, xc, yc; xc=0;
        int n, q;
        if(player==PLAYER1) {dir=1.0; i=25; color=CHECKER1_COLOR;}
        if(player==PLAYER2) {dir=-1.0; i=0; color=CHECKER2_COLOR;}
        n=abs(p->checker[i]); 
        if(n<=5) q=n;
        if(n>5) q=5;
        yc=dir*POINT_SIZE;
        for(int k=0; k<q; k++) {
            cdCanvasForeground(cv, color);
            wdCanvasSector(cv, xc, yc, CHECKER_SIZE, CHECKER_SIZE, 0, 360);
            cdCanvasForeground(cv, CHECKER_LINECOLOR);
            cdCanvasLineWidth(cv, CHECKER_LINEWIDTH);
            cdCanvasLineStyle(cv, CD_CONTINUOUS);
            wdCanvasArc(cv, xc, yc, CHECKER_SIZE, CHECKER_SIZE, 0, 360);
            yc += dir*CHECKER_SIZE;
        }
        if(n>5) {
            if(player==PLAYER1) cdCanvasForeground(cv, CHECKER1_TEXTCOLOR);
            if(player==PLAYER2) cdCanvasForeground(cv, CHECKER2_TEXTCOLOR);
            draw_number_checkers(xc, dir*(POINT_SIZE+4.*CHECKER_SIZE), n);
        }
    }

    draw_checker_onbar(PLAYER1);
    draw_checker_onbar(PLAYER2);

    xc = eps*(BOARD_WIDTH/2 -0.5*POINT_SIZE);
    for(int i=24; i>=19; i--) {
        yc = BOARD_HEIGHT/2 -0.5*CHECKER_SIZE;
        draw_checker_samepoint(xc, yc, i, -1);
        xc -= eps*POINT_SIZE;
    }

    xc = eps*-POINT_SIZE;
    for(int i=18; i>=13; i--) {
        yc = BOARD_HEIGHT/2 -0.5*CHECKER_SIZE;
        draw_checker_samepoint(xc, yc, i, -1);
        xc -= eps*POINT_SIZE;
    }

    xc = eps*(-BOARD_WIDTH/2+POINT_SIZE/2);
    for(int i=12; i>=7; i--) {
        yc = -BOARD_HEIGHT/2 +0.5*CHECKER_SIZE;
        draw_checker_samepoint(xc, yc, i, 1);
        xc += eps*POINT_SIZE;
    }

    xc = eps*POINT_SIZE;
    for(int i=6; i>=1; i--) {
        yc = -BOARD_HEIGHT/2 +0.5*CHECKER_SIZE;
        draw_checker_samepoint(xc, yc, i, 1);
        xc += eps*POINT_SIZE;
    }

}

void draw_canvas(cdCanvas* cv) {
    int i, w, h;
    int pip1=0, pip2=0;
    int off1=0, off2=0;
    cdCanvasActivate(cv);
    cdCanvasGetSize(cv, &w, &h, NULL, NULL);
    printf("canvas: %i, %i\n", w, h);
    cdCanvasBackground(cv, CD_WHITE);
    cdCanvasClear(cv);

    wdCanvasViewport(cv, 0, w-1, 0, h-1);

    double wd_h = BOARD_WITH_DECORATIONS_HEIGHT;
    double wd_w = (double) w* wd_h/(double) h;
    wdCanvasWindow(cv, -wd_w/2, wd_w/2, -wd_h/2, wd_h/2);

    compute_pipcount(pos_ptr, &pip1, &pip2);
    compute_checkeroff(pos_ptr, &off1, &off2);

    draw_board(cv);
    draw_checker(cv, pos_ptr, BOARD_DIRECTION);
    draw_cube(cv, pos_ptr->cube);
    draw_checkeroff(cv, off1, PLAYER1, BOARD_DIRECTION);
    draw_checkeroff(cv, off2, PLAYER2, BOARD_DIRECTION);
    if(is_pointletter_active) {
        draw_pointletter(cv, BOARD_DIRECTION, pos_ptr->cube);
    } else {
        draw_pointnumber(cv, BOARD_DIRECTION);
    }
    draw_score(cv, pos_ptr->p1_score, PLAYER1);
    draw_score(cv, pos_ptr->p2_score, PLAYER2);
    draw_pipcount(cv, pip1, PLAYER1);
    draw_pipcount(cv, pip2, PLAYER2);

    cdCanvasFlush(cv);
}



/************************ Prototypes **********************/

/* static int dlg_resize_cb(Ihandle*); */
static int canvas_map_cb(Ihandle*);
static int canvas_unmap_cb(Ihandle*);
static int canvas_action_cb(Ihandle*);
static int canvas_dropfiles_cb(Ihandle*);
static int canvas_motion_cb(Ihandle*);
static int canvas_wheel_cb(Ihandle*);
static int canvas_button_cb(Ihandle*, const int, const int,
        const int, const int, char*);
static int canvas_resize_cb(Ihandle*);
static int item_new_action_cb(void);
static int item_open_action_cb(void);
static int item_recent_action_cb(void);
static int item_save_action_cb(void);
static int item_saveas_action_cb(void);
static int item_import_action_cb(void);
static int item_export_action_cb(void);
static int item_properties_action_cb(void);
static int item_exit_action_cb(void);
static int item_undo_action_cb(void);
static int item_redo_action_cb(void);
static int item_copy_action_cb(void);
static int item_cut_action_cb(void);
static int item_paste_action_cb(void);
static int item_editmode_action_cb(void);
static int item_nextposition_action_cb(void);
static int item_prevposition_action_cb(void);
static int item_newposition_action_cb(void);
static int item_importposition_action_cb(void);
static int item_importpositionbybatch_action_cb(void);
static int item_newlibrary_action_cb(void);
static int item_deletelibrary_action_cb(void);
static int item_addtolibrary_action_cb(void);
static int item_importmatch_action_cb(void);
static int item_importmatchbybatch_action_cb(void);
static int item_matchlibrary_action_cb(void);
static int item_searchblunder_action_cb(void);
static int item_searchdice_action_cb(void);
static int item_searchcubedecision_action_cb(void);
static int item_searchscore_action_cb(void);
static int item_searchplayer_action_cb(void);
static int item_searchengine_action_cb(void);
static int item_searchmode_action_cb(void);
static int item_findpositionwithoutanalysis_action_cb(void);
static int item_preferences_action_cb(void);
static int item_helpmanual_action_cb(void);
static int item_userguide_action_cb(void);
static int item_tips_action_cb(void);
static int item_commandmodehelp_action_cb(void);
static int item_keyboardshortcuts_action_cb(void);
static int item_getinvolved_action_cb(void);
static int item_donate_action_cb(void);
static int item_about_action_cb(void);
static int set_visibility_off(Ihandle*);
static int set_visibility_on(Ihandle*);
static int toggle_visibility_cb(Ihandle*);
static int toggle_analysis_visibility_cb();
static int toggle_edit_visibility_cb();
static int toggle_searches_visibility_cb();
void error_callback(void);

/************************ Interface ***********************/

/* #define DEFAULT_SIZE "960x540" */
#define DEFAULT_SIZE "864x486"
/* #define DEFAULT_SIZE "800x450" */
#define DEFAULT_SPLIT_VALUE "700"
#define DEFAULT_SPLIT_MINMAX "800:2000"

Ihandle *dlg, *menu, *toolbar, *position, *split, *searches, *statusbar;
Ihandle *edit, *analysis, *canvas, *search, *matchlib;
Ihandle *search1, *search2, *search3;
Ihandle *hbox, *vbox, *lbl, *hspl, *vspl, *spl, *tabs, *txt;
bool is_searches_visible = false;

static Ihandle* create_menus(void)
{

    Ihandle *menu, *submenu_file, *submenu_edit,
            *submenu_position, *submenu_match,
            *submenu_search, *submenu_tool,
            *submenu_help;

    Ihandle *menu_file;
    Ihandle *item_new, *item_open, *item_recent;
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

    IupSetCallback(item_new, "ACTION", (Icallback) item_new_action_cb);
    IupSetCallback(item_open, "ACTION", (Icallback) item_open_action_cb);
    IupSetCallback(item_recent, "ACTION", (Icallback) item_recent_action_cb);
    IupSetCallback(item_import, "ACTION", (Icallback) item_import_action_cb);
    IupSetCallback(item_export, "ACTION", (Icallback) item_export_action_cb);
    IupSetCallback(item_save, "ACTION", (Icallback) item_save_action_cb);
    IupSetCallback(item_saveas, "ACTION", (Icallback) item_saveas_action_cb);
    IupSetCallback(item_properties, "ACTION", (Icallback) item_properties_action_cb);
    IupSetCallback(item_exit, "ACTION", (Icallback) item_exit_action_cb);
    IupSetCallback(item_undo, "ACTION", (Icallback) item_undo_action_cb);
    IupSetCallback(item_redo, "ACTION", (Icallback) item_redo_action_cb);
    IupSetCallback(item_cut, "ACTION", (Icallback) item_cut_action_cb);
    IupSetCallback(item_copy, "ACTION", (Icallback) item_copy_action_cb);
    IupSetCallback(item_paste, "ACTION", (Icallback) item_paste_action_cb);
    IupSetCallback(item_editmode, "ACTION", (Icallback) item_editmode_action_cb);
    IupSetCallback(item_next_position, "ACTION", (Icallback) item_nextposition_action_cb);
    IupSetCallback(item_prev_position, "ACTION", (Icallback) item_prevposition_action_cb);
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
    IupSetCallback(item_search_dice, "ACTION", (Icallback) item_searchdice_action_cb);
    IupSetCallback(item_search_cube, "ACTION", (Icallback) item_searchcubedecision_action_cb);
    IupSetCallback(item_search_score, "ACTION", (Icallback) item_searchscore_action_cb);
    IupSetCallback(item_search_player, "ACTION", (Icallback) item_searchplayer_action_cb);
    IupSetCallback(item_search_engine, "ACTION", (Icallback) item_searchengine_action_cb);
    IupSetCallback(item_searchmode, "ACTION", (Icallback) item_searchmode_action_cb);
    IupSetCallback(item_find_position_without_analysis, "ACTION", (Icallback) item_findpositionwithoutanalysis_action_cb);
    IupSetCallback(item_preferences, "ACTION", (Icallback) item_preferences_action_cb);
    IupSetCallback(item_manual, "ACTION", (Icallback) item_helpmanual_action_cb);
    IupSetCallback(item_userguide, "ACTION", (Icallback) item_userguide_action_cb);
    IupSetCallback(item_tips, "ACTION", (Icallback) item_tips_action_cb);
    IupSetCallback(item_cmdmode, "ACTION", (Icallback) item_commandmodehelp_action_cb);
    IupSetCallback(item_keyboard, "ACTION", (Icallback) item_keyboardshortcuts_action_cb);
    IupSetCallback(item_getinvolved, "ACTION", (Icallback) item_getinvolved_action_cb);
    IupSetCallback(item_donate, "ACTION", (Icallback) item_donate_action_cb);
    IupSetCallback(item_about, "ACTION", (Icallback) item_about_action_cb);


    return menu;

}

static Ihandle* create_toolbar(void)
{
    Ihandle *ih;
    Ihandle *btn_new, *btn_open, *btn_save, *btn_close, *btn_properties;
    Ihandle *btn_cut, *btn_copy, *btn_paste;
    Ihandle *btn_undo, *btn_redo;
    Ihandle *btn_prev, *btn_next;
    Ihandle *btn_blunder, *btn_dice, *btn_cube, *btn_score, *btn_player;
    Ihandle *btn_preferences;
    Ihandle *btn_manual;

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

    ih = IupHbox(
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

    IupSetAttribute(ih, "NAME", "TOOLBAR");
    IupSetAttribute(ih, "MARGIN", "5x5");
    IupSetAttribute(ih, "GAP", "2");

    IupSetCallback(btn_new, "ACTION", (Icallback) item_new_action_cb);
    IupSetCallback(btn_open, "ACTION", (Icallback) item_open_action_cb);
    IupSetCallback(btn_save, "ACTION", (Icallback) item_save_action_cb);
    IupSetCallback(btn_properties, "ACTION", (Icallback) item_properties_action_cb);
    IupSetCallback(btn_close, "ACTION", (Icallback) item_exit_action_cb);
    IupSetCallback(btn_undo, "ACTION", (Icallback) item_undo_action_cb);
    IupSetCallback(btn_redo, "ACTION", (Icallback) item_redo_action_cb);
    IupSetCallback(btn_cut, "ACTION", (Icallback) item_cut_action_cb);
    IupSetCallback(btn_copy, "ACTION", (Icallback) item_copy_action_cb);
    IupSetCallback(btn_paste, "ACTION", (Icallback) item_paste_action_cb);
    IupSetCallback(btn_next, "ACTION", (Icallback) item_nextposition_action_cb);
    IupSetCallback(btn_prev, "ACTION", (Icallback) item_prevposition_action_cb);
    IupSetCallback(btn_blunder, "ACTION", (Icallback) item_searchblunder_action_cb);
    IupSetCallback(btn_dice, "ACTION", (Icallback) item_searchdice_action_cb);
    IupSetCallback(btn_cube, "ACTION", (Icallback) item_searchcubedecision_action_cb);
    IupSetCallback(btn_score, "ACTION", (Icallback) item_searchscore_action_cb);
    IupSetCallback(btn_player, "ACTION", (Icallback) item_searchplayer_action_cb);
    IupSetCallback(btn_preferences, "ACTION", (Icallback) item_preferences_action_cb);
    IupSetCallback(btn_manual, "ACTION", (Icallback) item_helpmanual_action_cb);

    return ih;
}

static Ihandle* create_statusbar(void)
{
    Ihandle *ih;

    ih = IupLabel("NORMAL MODE");
    IupSetAttribute(ih, "NAME", "STATUSBAR");
    IupSetAttribute(ih, "EXPAND", "HORIZONTAL");
    IupSetAttribute(ih, "PADDIND", "10x5");

    return ih;
}

static Ihandle* create_canvas(void)
{
    Ihandle *ih;
    ih = IupCanvas(NULL);
    IupSetAttribute(ih, "NAME", "CANVAS");
    /* IupSetAttribute(ih, "SIZE", "300x200"); */
    /* IupSetAttribute(ih, "MINSIZE", "600x200"); */
    /* IupSetAttribute(ih, "MAXSIZE", "600x300"); */
    IupSetAttribute(ih, "BGCOLOR", "255 255 255");
    IupSetAttribute(ih, "BORDER", "YES");
    /* IupSetAttribute(ih, "DRAWSIZE", "200x300"); */
    IupSetAttribute(ih, "EXPAND", "YES");
    IupSetCallback(ih, "MAP_CB", (Icallback)canvas_map_cb);
    IupSetCallback(ih, "UNMAP_CB", (Icallback)canvas_unmap_cb);
    IupSetCallback(ih, "ACTION", (Icallback)canvas_action_cb);
    IupSetCallback(ih, "DROPFILES_CB", (Icallback)canvas_dropfiles_cb);
    IupSetCallback(ih, "MOTION_CB", (Icallback)canvas_motion_cb);
    IupSetCallback(ih, "WHEEL_CB", (Icallback)canvas_wheel_cb);
    IupSetCallback(ih, "BUTTON_CB", (Icallback)canvas_button_cb);
    IupSetCallback(ih, "RESIZE_CB", (Icallback)canvas_resize_cb);

    return ih;
}

static Ihandle* create_analysis(void)
{
    Ihandle *ih;

    ih = IupLabel("ANALYSIS HERE");
    IupSetAttribute(ih, "VISIBLE", "NO");
    IupSetAttribute(ih, "FLOATING", "YES");
    /* IupSetAttribute(ih, "ORIENTATION", "VERTICAL"); */
    /* IupSetAttribute(exp_analysis, "ORIENTATION", "VERTICAL"); */
    /* IupSetAttribute(exp_analysis, "TITLE", "MyMenu"); */
    /* IupSetAttribute(exp_analysis, "STATE", "CLOSE"); */
    /* IupSetAttribute(exp_analysis, "GAP", "2"); */

    return ih;
}

static Ihandle* create_search(void)
{
    Ihandle *ih;
    ih = IupCells();
    IupSetAttribute(ih, "BOXED", "YES");

    return ih;
}

static Ihandle* create_edit(void)
{
    Ihandle *ih;

    ih = IupLabel("Edit Panel");
    IupSetAttribute(ih, "NAME", "EDIT");
    IupSetAttribute(ih, "EXPAND", "YES");
    IupSetAttribute(ih, "VISIBLE", "NO");
    IupSetAttribute(ih, "FLOATING", "YES");

    return ih;
}

static Ihandle* create_library(void)
{
    Ihandle *ih;

    return ih;
}

static Ihandle* create_matchlibrary(void)
{
    Ihandle *ih;

    return ih;
}

static Ihandle* create_searches(void)
{
    Ihandle *ih;

    search1 = create_search();
    IupSetAttribute(search1, "TABTITLE", "Search1 Position");

    search2 = create_search();
    IupSetAttribute(search2, "TABTITLE", "search2 Position");

    search3 = create_search();
    IupSetAttribute(search3, "TABTITLE", "search3 Position");

    ih = IupTabs(search1, search2, search3, NULL);
    IupSetAttribute(ih, "VISIBLE", "NO");
    IupSetAttribute(ih, "FLOATING", "YES");

    return ih;
}

/*************** Keyboard Shortcuts ***********************/

// FAIRE CAS PLUS DE 5 CHECKERS SUR POINT OU BAR !!!!
static void set_keyboard_shortcuts()
{
    IupSetCallback(dlg, "K_cN", (Icallback) item_new_action_cb);
    IupSetCallback(dlg, "K_cO", (Icallback) item_open_action_cb);
    IupSetCallback(dlg, "K_cS", (Icallback) item_save_action_cb);
    IupSetCallback(dlg, "K_cQ", (Icallback) item_exit_action_cb);
    IupSetCallback(dlg, "K_cZ", (Icallback) item_undo_action_cb);
    IupSetCallback(dlg, "K_cE", (Icallback) toggle_edit_visibility_cb);
    IupSetCallback(dlg, "K_cI", (Icallback) toggle_analysis_visibility_cb);
    IupSetCallback(dlg, "K_cF", (Icallback) toggle_searches_visibility_cb);
}

/************************ Callbacks ***********************/

static int canvas_map_cb(Ihandle* ih)
{
    cdv = cdCreateCanvas(CD_IUP, canvas);
    return IUP_DEFAULT;
}

static int canvas_unmap_cb(Ihandle* ih)
{
    cdKillCanvas(cdv);
    return IUP_DEFAULT;
}

static int canvas_action_cb(Ihandle* ih)
{
    draw_canvas(cdv);
    return IUP_DEFAULT;
}

static int canvas_dropfiles_cb(Ihandle* ih)
{
    error_callback();
    return IUP_DEFAULT;
}

static int canvas_motion_cb(Ihandle* ih)
{
    /* error_callback(); */
    return IUP_DEFAULT;
}

static int canvas_wheel_cb(Ihandle* ih)
{
    error_callback();
    return IUP_DEFAULT;
}

static int canvas_button_cb(Ihandle* ih, const int button,
        const int pressed, const int x, const int y, char* status)
{
    double xw, yw, xwp, ywp;
    int y2, y2p;
    int dir, player, ix, iy, ixp, iyp, i, ip, i1, i2, iyn;
    bool mouse_hold;
    bool is_in_left, is_in_right, is_in_up, is_in_down, is_on_bar, is_in_center;
    bool is_in_uplabel, is_in_downlabel, is_in_board; 
    bool is_in_cube, is_cube_in_center, is_cube_up, is_cube_down, 
         is_in_cube_positions;
    bool is_on_score1, is_on_score2;

    mouse_hold=false;

    if(BOARD_DIRECTION==1) dir=1;
    if(BOARD_DIRECTION!=1) dir=-1;

    // canvas and world have inverted y axis...
    y2 = cdCanvasInvertYAxis(cdv, y);
    wdCanvasCanvas2World(cdv, x, y2, &xw, &yw);
    ix = round(xw/POINT_SIZE);
    iy = round(yw/POINT_SIZE);
    printf("ix: %i\niy: %i\n", ix, iy);

    // for previous mouse state if clicked
    if(mouse.pressed==1) {
        y2p = cdCanvasInvertYAxis(cdv, mouse.y);
        wdCanvasCanvas2World(cdv, mouse.x, y2p, &xwp, &ywp);
        ixp = round(xwp/POINT_SIZE);
        iyp = round(ywp/POINT_SIZE);
        /* printf("PREVIOUS MOUSE STATE CLICKED\n"); */
        /* printf("ixp: %i\niyp: %i\n", ixp, iyp); */
        if(ix!=ixp || iy!=iyp ) mouse_hold=true;
    }

    // labels (number or point) are in the board
    is_in_board = abs(ix)<=6 && abs(iy)<=6;
    is_in_uplabel = is_in_board && iy==6;
    is_in_downlabel = is_in_board && iy==-6;
    is_in_left = ix<0 && ix>=-6;
    is_in_up = iy>0 && iy<=6;
    is_in_down = iy<0 && iy>=-6;
    is_in_right = ix>0 && ix<=6;
    is_on_bar = is_in_board && ix==0;
    is_in_center = ix==0 && iy==0;
    is_cube_in_center = (xw>=CUBE_XPOS) && (xw<=CUBE_XPOS+CUBE_SIZE)
        && (yw>=CUBE_YPOS_CENTER) && (yw<=CUBE_YPOS_CENTER+CUBE_SIZE);
    is_cube_down = (xw>=CUBE_XPOS) && (xw<=CUBE_XPOS+CUBE_SIZE)
        && (yw>=CUBE_YPOS_DOWN) && (yw<=CUBE_YPOS_DOWN+CUBE_SIZE);
    is_cube_up = (xw>=CUBE_XPOS) && (xw<=CUBE_XPOS+CUBE_SIZE)
        && (yw>=CUBE_YPOS_UP) && (yw<=CUBE_YPOS_UP+CUBE_SIZE);
    is_in_cube_positions = is_cube_in_center || is_cube_down || is_cube_up;
    is_in_cube = is_cube_in_center;
    if(pos_ptr->cube>0) is_in_cube = is_cube_down;
    if(pos_ptr->cube<0) is_in_cube = is_cube_up;
    is_on_score1 = (xw>=SCORE_XPOS-.5*POINT_SIZE) &&
        (yw<=SCORE_YPOS_DOWN+1.5*POINT_SIZE);
    is_on_score2 = (xw>=SCORE_XPOS-1.*POINT_SIZE) &&
        (yw>=SCORE_YPOS_UP-1.*POINT_SIZE);

    if(button==IUP_BUTTON1) player=1;
    if(button==IUP_BUTTON3) player=-1;

    int fill_point(const int n) {
        return player*(6-abs(n)); }

    int find_point_index(const int ix, const int iy) {
        int i;
        is_in_left = ix<0 && ix>=-6;
        is_in_up = iy>0 && iy<=6;
        is_in_down = iy<0 && iy>=-6;
        is_in_right = ix>0 && ix<=6;
        if(is_in_left) {
            if(is_in_up) {
                if(BOARD_DIRECTION==1) i=19+ix;
                if(BOARD_DIRECTION!=1) i=18-ix;
            } else if(is_in_down) {
                if(BOARD_DIRECTION==1) i=6-ix;
                if(BOARD_DIRECTION!=1) i=7+ix;
            }
        } else if(is_in_right) {
            if(is_in_up) {
                if(BOARD_DIRECTION==1) i=18+ix;
                if(BOARD_DIRECTION!=1) i=19-ix;
            } else if(is_in_down) {
                if(BOARD_DIRECTION==1) i=7-ix;
                if(BOARD_DIRECTION!=1) i=6+ix;
            }
        }
        return i;
    }

    if(!pressed){
        if(is_in_board && !is_on_bar) {
            i=find_point_index(ix, iy); 
            if(abs(iy)==1 && abs(pos_ptr->checker[i])>=5) {
                pos_ptr->checker[i] += player;
            } else { pos_ptr->checker[i] = fill_point(iy); }
        } else if(is_on_bar) {
            if(is_in_up) {
                if(!is_in_uplabel) {
                    if(abs(iy)==5 && abs(pos_ptr->checker[25])>=5) {
                        pos_ptr->checker[25] -= 1;
                    } else { pos_ptr->checker[25] = (-iy); }
                } else {pos_ptr->checker[25] = 0; }
            } else if(is_in_down) {
                if(!is_in_downlabel) {
                    if(abs(iy)==5 && abs(pos_ptr->checker[0])>=5) {
                        pos_ptr->checker[0] += 1;
                    } else { pos_ptr->checker[0] = (-iy); }
                } else { pos_ptr->checker[0] = 0; }
            } else if(is_in_center) {
                pos_ptr->checker[25] = 0;
                pos_ptr->checker[0] = 0;
            } else { printf("ERROR! Cas impossible!\n"); }
        }
    }

    if(mouse_hold){
        ip=find_point_index(ixp, iyp);
        i1=fmin(i,ip);
        i2=fmax(i,ip);
        iyn=fmin(abs(iy), abs(iyp));
        if(iyn==0) iyn=1;
        for(int k=i1; k<=i2; k++) {
            pos_ptr->checker[k] = fill_point(iyn);
        }
    }

    if(!pressed){
        if(is_in_cube && button==IUP_BUTTON1)
            pos_ptr->cube +=1;
        if(is_in_cube && button==IUP_BUTTON3)
            pos_ptr->cube -=1;
    }

    if(!pressed){
        if(is_on_score1) {
            if(button==IUP_BUTTON1) pos_ptr->p1_score -=1;
            if(button==IUP_BUTTON3) pos_ptr->p1_score +=1;
            if(pos_ptr->p1_score<-1) pos_ptr->p1_score=-1;
        }
        if(is_on_score2) {
            if(button==IUP_BUTTON1) pos_ptr->p2_score -=1;
            if(button==IUP_BUTTON3) pos_ptr->p2_score +=1;
            if(pos_ptr->p2_score<-1) pos_ptr->p2_score=-1;
        }
    }

    if(iup_isdouble(status)){
        if(!is_in_board && !is_on_score1 && !is_on_score2
                && !is_in_cube_positions) {
            for(int i=0; i<26; i++) {
                pos_ptr->checker[i]=0;
            }
        }
    }

    if(!pressed) draw_canvas(cdv);

    mouse.button = button;
    mouse.pressed = pressed;
    mouse.x = x;
    mouse.y = y;
    mouse.status = status;
    mouse_print(mouse);
    return IUP_DEFAULT;
}

/* static int dlg_resize_cb(Ihandle* ih) */
/* { */
/*     IupRefresh(ih); */
/*     return IUP_DEFAULT; */
/* } */

static int canvas_resize_cb(Ihandle* ih)
{
    cdCanvasActivate(cdv);
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
            printf("IupFileDlg: Operation Canceled\n");
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
            printf("IupFileDlg: Operation Canceled");
            return 1;
            break; 
    }

    IupDestroy(filedlg);
    return IUP_DEFAULT;

}

static int item_recent_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_import_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_export_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_properties_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}


static int item_save_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_saveas_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
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
    return IUP_DEFAULT;
}

static int item_redo_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_copy_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_cut_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_paste_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_editmode_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_nextposition_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_prevposition_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_newposition_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_importposition_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_importpositionbybatch_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_newlibrary_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_deletelibrary_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_addtolibrary_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_importmatch_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_importmatchbybatch_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_matchlibrary_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_searchblunder_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_searchdice_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_searchcubedecision_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_searchscore_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_searchplayer_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_searchengine_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_searchmode_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_findpositionwithoutanalysis_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_preferences_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_helpmanual_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_userguide_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_tips_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_commandmodehelp_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_keyboardshortcuts_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_getinvolved_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_donate_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_about_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
    
}

static int set_visibility_off(Ihandle* ih)
{
    IupSetAttribute(ih, "VISIBLE", "NO");
    IupSetAttribute(ih, "FLOATING", "YES");
    return IUP_DEFAULT;
}

static int set_visibility_on(Ihandle* ih)
{
    IupSetAttribute(ih, "VISIBLE", "YES");
    IupSetAttribute(ih, "FLOATING", "NO");
    return IUP_DEFAULT;
}

static int toggle_edit_visibility_cb()
{
    toggle_visibility_cb(edit);
    return IUP_DEFAULT;
}

static int toggle_analysis_visibility_cb()
{
    toggle_visibility_cb(analysis);
    return IUP_DEFAULT;
}

static int toggle_searches_visibility_cb()
{
    toggle_visibility_cb(searches);
    char* att = IupGetAttribute(searches, "VISIBLE");
    if(strcmp(att,"NO") == 0) {
        IupSetAttribute(split, "VALUE", "1000");
    } else if (strcmp(att,"YES") == 0) {
        IupSetAttribute(split, "VALUE", DEFAULT_SPLIT_VALUE);
    }
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int toggle_visibility_cb(Ihandle* ih)
{
    char* att = IupGetAttribute(ih, "VISIBLE");
    if(strcmp(att,"NO")==0) {
        printf("display panel\n");
        IupSetAttribute(ih, "VISIBLE", "YES");
        IupSetAttribute(ih, "FLOATING", "NO");
    } else if (strcmp(att, "YES")==0) {
        printf("hide panel\n");
        IupSetAttribute(ih, "VISIBLE", "NO");
        IupSetAttribute(ih, "FLOATING", "YES");
    } else {
        printf("panel_ih_visible_cb: Impossible case.\n");
    }
    IupRefresh(ih);
    return IUP_DEFAULT;
}

void error_callback(void)
{
    IupMessage("Callback Error", "Functionality not implemented yet!");
}

/************************ Main ****************************/
int main(int argc, char **argv)
{
    int err;

    pos = POS_DEFAULT;
    /* pos = POS_VOID; */
    pos_ptr = &pos;

    /* err = str_to_pos("-1,-1:(a-f)", pos_ptr); */
    /* err = str_to_pos("0,3:(a-f)", pos_ptr); */
    /* err = str_to_pos("1,3:(a-f)", pos_ptr); */
    /* err = str_to_pos("(a-f)", pos_ptr); */
    /* err = str_to_pos("(f-a)", pos_ptr); */
    /* err = str_to_pos("31,12:Z2y1(e-aX)F3(mnl)t-pO4Y3", pos_ptr); */
    /* err = str_to_pos("(SUmLhgfDc)AS2m2TWQRgf2", pos_ptr); */
    /* printf("str2pos err: %i\n", err); */
    /* pos_print(pos_ptr); */
    /* pos_ptr->checker[24] = 25; */

    /* char* ctest; */
    /* ctest= pos_to_str(&POS_DEFAULT); */
    /* printf("ctest: %s\n", ctest); */
    /* ctest= pos_to_str_paren(&POS_DEFAULT); */
    /* printf("ctest2: %s\n", ctest); */
    /* free(ctest); */

    IupOpen(&argc, &argv);
    IupControlsOpen();
    IupImageLibOpen();
    IupSetLanguage("ENGLISH");

    menu = create_menus();
    toolbar = create_toolbar();

    canvas = create_canvas();
    analysis = create_analysis();
    edit = create_edit();
    position = IupHbox(analysis, edit, canvas, NULL);

    searches = create_searches();
    statusbar = create_statusbar();

    split = IupSplit(position, searches);
    IupSetAttribute(split, "ORIENTATION", "HORIZONTAL");
    IupSetAttribute(split, "VALUE", "1000");
    /* IupSetAttribute(split, "MINMAX", DEFAULT_SPLIT_MINMAX); */
    /* IupSetAttribute(split, "AUTOHIDE", "YES"); */

    vbox = IupVbox(toolbar, split, statusbar, NULL);
    IupSetAttribute(vbox, "NMARGIN", "10x10");
    IupSetAttribute(vbox, "GAP", "10");

    dlg = IupDialog(vbox);
    IupSetAttribute(dlg, "TITLE", "blunderDB");
    IupSetAttribute(dlg, "SIZE", DEFAULT_SIZE);
    IupSetAttribute(dlg, "SHRINK", "YES");
    IupSetAttribute(dlg, "MENU", "menu");

    set_keyboard_shortcuts();

    IupShowXY(dlg, IUP_CENTER, IUP_CENTER);

    IupMainLoop();

    db_close(db);
    IupClose();

    return EXIT_SUCCESS;
}
