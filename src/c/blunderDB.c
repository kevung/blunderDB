#include <ctype.h>
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

#define P1_POINT_LABEL "*abcdefghijklmnopqrstuvwxyz"
#define P2_POINT_LABEL "YABCDEFGHIJKLMNOPQRSTUVWX*Z"

typedef struct
{
    int checker[26];
    int cube;
    int is_crawford;
    int p1_score;
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
    .is_crawford = 0,
    .p1_score = 0,
    .p2_score = 0,
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
    .is_crawford = 0,
    .p1_score = 0,
    .p2_score = 0,
    .dice = {0, 0},
    .is_double = 0,
    .is_take = 0,
    .is_on_roll = 0,
};

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
    printf("is_crawford: %i\n", p->is_crawford);
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

char* pos_to_str(const POSITION* p)
{
    const char p1[27] = P1_POINT_LABEL;
    const char p2[27] = P2_POINT_LABEL;
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

char* pos_to_str_paren(const POSITION* p)
{
    const char p1[27] = P1_POINT_LABEL;
    const char p2[27] = P2_POINT_LABEL;
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


int str_to_pos(const char* s, POSITION* pos)
{
    const char p1[27] = P1_POINT_LABEL;
    const char p2[27] = P2_POINT_LABEL;
    int has_score, i_score = 0;
    char s_p1_score[5], s_p2_score[5];
    s_p1_score[0] = '\0';
    s_p2_score[0] = '\0';
    int i, j = 0;
    int len = strlen(s);
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
            if(!isdigit(s[i])) return 0; //fail
            s_p1_score[i] = s[i];
            s_p1_score[i+1] = '\0';
        }
        for(int i=j+1; i<i_score; i++)
        {
            if(!isdigit(s[i])) return 0; //fail
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
#define BOARD_WITH_DECORATIONS_HEIGHT BOARD_HEIGHT+2*POINT_SIZE
#define POINT_SIZE BOARD_WIDTH/13
#define CHECKER_SIZE POINT_SIZE
#define BAR_WIDTH CHECKER_SIZE
#define BOARD_COLOR CD_BLACK
#define BOARD_LINEWIDTH 5
#define TRIANGLE_LINEWIDTH 2
#define TRIANGLE_LINECOLOR CD_BLACK
#define CUBE_LINEWIDTH 5
#define CUBE_TEXTLINEWIDTH 3
#define CUBE_LINECOLOR CD_BLACK
#define CUBE_SIZE 1.1*CHECKER_SIZE
#define CUBE_FONT "Times"
#define CUBE_FONTSIZE 28
#define CUBE_STYLE CD_PLAIN
#define CUBE_XPOS -BOARD_WIDTH/2 -1.5*POINT_SIZE
#define CUBE_YPOS_CENTER 0
#define CUBE_YPOS_UP BOARD_HEIGHT/2 - CUBE_SIZE
#define CUBE_YPOS_DOWN -BOARD_HEIGHT/2
#define POINTNUMBER_FONT "Times"
#define POINTNUMBER_FONTSIZE 20
#define POINTNUMBER_STYLE CD_PLAIN
#define POINTNUMBER_LINECOLOR CD_BLACK
#define POINTNUMBER_YPOS_UP BOARD_HEIGHT/2+POINT_SIZE/2
#define POINTNUMBER_YPOS_DOWN -(POINTNUMBER_YPOS_UP)
#define SCORE_XPOS BOARD_WIDTH/2+1.7*POINT_SIZE
#define SCORE_YPOS_UP BOARD_HEIGHT/2+0.1*POINT_SIZE
#define SCORE_YPOS_DOWN -(SCORE_YPOS_UP)
#define SCORE_FONT "Times"
#define SCORE_FONTSIZE 20
/* #define SCORE_STYLE CD_PLAIN */
/* #define SCORE_STYLE CD_ITALIC */
#define SCORE_STYLE CD_BOLD
#define SCORE_LINECOLOR CD_BLACK
#define CHECKEROFF_XPOS BOARD_WIDTH/2+POINT_SIZE
#define CHECKEROFF_YPOS_UP 3*CHECKER_SIZE
#define CHECKEROFF_YPOS_DOWN -(CHECKEROFF_YPOS_UP)
#define CHECKEROFF_FONT "Times"
#define CHECKEROFF_FONTSIZE 15
/* #define CHECKEROFF_STYLE CD_PLAIN */
#define CHECKEROFF_STYLE CD_ITALIC
#define CHECKEROFF_LINECOLOR CD_BLACK
#define PIPCOUNT_XPOS -BOARD_WIDTH/2-2.0*POINT_SIZE
#define PIPCOUNT_YPOS_UP POINTNUMBER_YPOS_UP
#define PIPCOUNT_YPOS_DOWN -(PIPCOUNT_YPOS_UP)
#define PIPCOUNT_FONT "Times"
#define PIPCOUNT_FONTSIZE 15
/* #define PIPCOUNT_STYLE CD_PLAIN */
#define PIPCOUNT_STYLE CD_PLAIN
#define PIPCOUNT_LINECOLOR CD_BLACK

cdCanvas *cdv = NULL;
cdCanvas *db_cdv = NULL;

void draw_triangle(cdCanvas *cv, double x, double y, double up){
    cdCanvasForeground(cdv, CD_WHITE);
    cdCanvasBegin(cdv, CD_FILL);
    wdCanvasVertex(cdv, x, y);
    wdCanvasVertex(cdv, x+POINT_SIZE, y);
    wdCanvasVertex(cdv, x+POINT_SIZE/2, y + ((double) up)*5*POINT_SIZE);
    cdCanvasEnd(cdv);

    cdCanvasLineWidth(cdv, TRIANGLE_LINEWIDTH);
    cdCanvasForeground(cdv, TRIANGLE_LINECOLOR);
    cdCanvasLineStyle(cdv, CD_CONTINUOUS);
    cdCanvasBegin(cdv, CD_CLOSED_LINES);
    wdCanvasVertex(cdv, x, y);
    wdCanvasVertex(cdv, x+POINT_SIZE, y);
    wdCanvasVertex(cdv, x+POINT_SIZE/2, y + ((double) up)*5*POINT_SIZE);
    cdCanvasEnd(cdv);
}

char* cubeText(int value) {
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

void draw_cube(cdCanvas *cv, int value){
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
        draw_triangle(cv, x, y, 1);
        draw_triangle(cv, x+POINT_SIZE, -y, -1);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2, y, 1);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2+POINT_SIZE, -y, -1);
    }

    cdCanvasHatch(cv, CD_HORIZONTAL);
    for(int i=0; i<3; i++){
        double x = -BOARD_WIDTH/2 +((double) i)*2*POINT_SIZE;
        double y = -BOARD_HEIGHT/2 +BOARD_HEIGHT;
        draw_triangle(cv, x, y, -1);
        draw_triangle(cv, x+POINT_SIZE, -y, 1);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2, y, -1);
        draw_triangle(cv, x+(BOARD_WIDTH+BAR_WIDTH)/2+POINT_SIZE, -y, 1);
    }

    cdCanvasForeground(cv, BOARD_COLOR);
    cdCanvasLineWidth(cv, BOARD_LINEWIDTH);
    cdCanvasLineStyle(cv, CD_CONTINUOUS);
    wdCanvasRect(cv, -BOARD_WIDTH/2, BOARD_WIDTH/2,
            -BOARD_HEIGHT/2, BOARD_HEIGHT/2);
    cdCanvasLineWidth(cv, BOARD_LINEWIDTH);
    wdCanvasRect(cv, -BAR_WIDTH/2, BAR_WIDTH/2,
            -BOARD_HEIGHT/2, BOARD_HEIGHT/2);
}

void draw_pointnumber(cdCanvas* cv, int orientation) {
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

void draw_pointletter(cdCanvas* cv, int orientation, int cubevalue) {
    const char p1[27] = P1_POINT_LABEL;
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

void draw_score(cdCanvas* cv, int score, int crawford, int player){
    char t[20];
    cdCanvasForeground(cv, SCORE_LINECOLOR);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, SCORE_FONT, SCORE_STYLE, SCORE_FONTSIZE);
    sprintf(t, "%d", score);
    strcat(t, " away");
    if(score==1 && crawford==1) strcat(t, "\ncrawford");
    if(score==1 && crawford!=1) strcat(t, "\npost\ncrawford");
    if(player>0) {
        wdCanvasText(cv, SCORE_XPOS, SCORE_YPOS_DOWN, t);
    } else {
        wdCanvasText(cv, SCORE_XPOS, SCORE_YPOS_UP, t);
    }
}

void draw_checkeroff(cdCanvas* cv, int nb_off, int player, int orientation){
    char t[20], t2[3];
    cdCanvasForeground(cv, CHECKEROFF_LINECOLOR);
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

void draw_pipcount(cdCanvas* cv, int pip, int player){
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
/************************ Prototypes **********************/

/* static int dlg_resize_cb(Ihandle* ih); */
static int canvas_action_cb(Ihandle* ih);
static int canvas_dropfiles_cb(Ihandle* ih);
static int canvas_motion_cb(Ihandle* ih);
static int canvas_wheel_cb(Ihandle* ih);
static int canvas_resize_cb(Ihandle* ih);
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
static int set_visibility_off(Ihandle* ih);
static int set_visibility_on(Ihandle* ih);
static int toggle_visibility_cb(Ihandle* ih);
static int toggle_analysis_visibility_cb();
static int toggle_edit_visibility_cb();
static int toggle_searches_visibility_cb();
void error_callback(void);

/************************ Interface ***********************/

#define DEFAULT_SIZE "800x600"
#define DEFAULT_SPLIT_VALUE "800"
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
    cdv = cdCreateCanvas(CD_IUP, ih);
    IupSetAttribute(ih, "NAME", "CANVAS");
    /* IupSetAttribute(ih, "SIZE", "300x200"); */
    /* IupSetAttribute(ih, "MINSIZE", "600x200"); */
    /* IupSetAttribute(ih, "MAXSIZE", "600x300"); */
    IupSetAttribute(ih, "BGCOLOR", "255 255 255");
    IupSetAttribute(ih, "BORDER", "YES");
    /* IupSetAttribute(ih, "DRAWSIZE", "200x300"); */
    IupSetAttribute(ih, "EXPAND", "YES");
    IupSetCallback(ih, "ACTION", (Icallback)canvas_action_cb);
    IupSetCallback(ih, "DROPFILES_CB", (Icallback)canvas_dropfiles_cb);
    IupSetCallback(ih, "MOTION_CB", (Icallback)canvas_motion_cb);
    IupSetCallback(ih, "WHEEL_CB", (Icallback)canvas_wheel_cb);
    /* IupSetCallback(ih, "RESIZE_CB", (Icallback)canvas_resize_cb); */

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

    return ih;
}

static Ihandle* create_position(void)
{

    Ihandle *ih, *spl;

    canvas = create_canvas();
    analysis = create_analysis();
    edit = create_edit();
    ih = IupHbox(analysis, edit, canvas, NULL);
    /* ih = IupHbox(edit, canvas, NULL); */
    /* IupSetAttribute(ih, "ORIENTATION", "VERTICAL"); */
    /* IupSetAttribute(ih, "VALUE", "0"); */
    /* IupSetAttribute(spl, "MINMAX", "0:1000"); */

    return ih;
}

/*************** Keyboard Shortcuts ***********************/

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

static int canvas_action_cb(Ihandle* ih)
{
    int i, w, h;
    cdv = cdCreateCanvas(CD_IUP, ih);
    cdCanvasGetSize(cdv, &w, &h, NULL, NULL);
    printf("canvas: %i, %i\n", w, h);
    cdCanvasBackground(cdv, CD_WHITE);
    cdCanvasClear(cdv);

    wdCanvasViewport(cdv, 0, w-1, 0, h-1);

    double wd_h = BOARD_WITH_DECORATIONS_HEIGHT;
    double wd_w = (double) w* wd_h/(double) h;
    wdCanvasWindow(cdv, -wd_w/2, wd_w/2, -wd_h/2, wd_h/2);


    int _cube = 1;
    int _orig = 1;
    draw_board(cdv);
    draw_cube(cdv, _cube);
    draw_pointnumber(cdv, _orig);
    /* draw_pointletter(cdv, _orig, _cube); */
    draw_score(cdv, 1, -1, 1);
    draw_score(cdv, 5, 0, -1);
    draw_checkeroff(cdv, 4, 1, _orig);
    draw_checkeroff(cdv, 6, -1, _orig);
    draw_pipcount(cdv, 167, 1);
    draw_pipcount(cdv, 132, -2);

    cdCanvasFlush(cdv);

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

    /* IupUnmap(split); */
    /* IupDetach(split); */
    /* IupAppend(vbox, canvas); */
    /* /1* IupReparent(canvas, vbox, statusbar); *1/ */
    /* /1* IupMap(canvas); *1/ */
    /* IupMap(canvas); */
    /* IupMap(vbox); */
    /* IupMap(dlg); */
    /* IupRefresh(dlg); */

    toggle_visibility_cb(searches);

    char* att = IupGetAttribute(searches, "VISIBLE");
    if(strcmp(att,"NO") == 0)
    {
        IupSetAttribute(split, "VALUE", "1000");
    } else if (strcmp(att,"YES") == 0)
    {
        IupSetAttribute(split, "VALUE", DEFAULT_SPLIT_VALUE);
    }
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int toggle_visibility_cb(Ihandle* ih)
{
    /* Ihandle *child; */

    /* int n = IupGetChildCount(ih); */
    /* printf("children: %i\n", n); */
    /* for(int i=0; i<n; i++) */
    /* { */
    /*     toggle_visibility_cb(IupGetChild(ih, i)); */
    /* } */


    char* att = IupGetAttribute(ih, "VISIBLE");

    if(strcmp(att,"NO")==0)
    {
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
  IupOpen(&argc, &argv);
  IupControlsOpen();
  IupImageLibOpen();
  IupSetLanguage("ENGLISH");

  /* char* ctest; */
  /* ctest= pos_to_str(&POS_DEFAULT); */
  /* printf("ctest: %s\n", ctest); */
  /* ctest= pos_to_str_paren(&POS_DEFAULT); */
  /* printf("ctest2: %s\n", ctest); */
  /* free(ctest); */

  POSITION pos = POS_VOID;
  POSITION* pos_ptr = &pos;
  /* pos_print(pos_ptr); */
  char* ctest = "31,12:Z11y1(e-aX)F3(mnl)t-pO4Y3";
  printf("pos: %s\n", ctest);
  str_to_pos(ctest, pos_ptr);
  pos_print(pos_ptr);


  menu = create_menus();
  toolbar = create_toolbar();
  position = create_position();
  searches = create_searches();
  statusbar = create_statusbar();

  split = IupSplit(position, searches);
  IupSetAttribute(split, "ORIENTATION", "HORIZONTAL");
  IupSetAttribute(split, "VALUE", DEFAULT_SPLIT_VALUE);
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

  /* IupSetCallback(dlg, "RESIZE_CB", (Icallback)dlg_resize_cb); */
  set_keyboard_shortcuts();

  IupShowXY(dlg, IUP_CENTER, IUP_CENTER);

  IupMainLoop();

  db_close(db);
  IupClose();

  return EXIT_SUCCESS;
}
