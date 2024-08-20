#include <ctype.h>
#include <stdbool.h>
#include <math.h>
#include <stdbool.h>
#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>
#include <iup.h>
#include <iupdraw.h>
#include <iupcontrols.h>
#include <cd.h>
#include <cdiup.h>
#include <wd.h>
#include <sqlite3.h>

/* Main sections: */
/* - Prototypes, */ 
/* - Data, */
/* - Database, */
/* - Interface, */
/* - Drawing, */
/* - Keyboard Shortcuts, */ 
/* - Callbacks */

/************************ Prototypes **********************/
// BEGIN Prototypes

/* static int dlg_resize_cb(Ihandle*); */
static int canvas_map_cb(Ihandle*);
static int canvas_unmap_cb(Ihandle*);
static int canvas_action_cb(Ihandle*);
static int canvas_dropfiles_cb(Ihandle*, const char*, int, int, int);
static int canvas_motion_cb(Ihandle*);
static int canvas_wheel_cb(Ihandle*, float, int, int, char*);
static int canvas_button_cb(Ihandle*, const int, const int,
        const int, const int, char*);
static int canvas_resize_cb(Ihandle*);
static int item_new_action_cb(void);
static int item_open_action_cb(void);
static int item_save_action_cb(void);
static int item_saveas_action_cb(void);
static int item_import_action_cb(void);
static int item_export_action_cb(void);
static int item_properties_action_cb(void);
static int item_exit_action_cb(void);
static int item_copy_action_cb(void);
static int item_paste_action_cb(void);
static int item_editmode_action_cb(void);
static int item_nextposition_action_cb(void);
static int item_prevposition_action_cb(void);
static int item_newposition_action_cb(void);
static int item_updateposition_action_cb(void);
static int item_importposition_action_cb(void);
static int item_importpositionbybatch_action_cb(void);
static int item_newlibrary_action_cb(void);
static int item_deletelibrary_action_cb(void);
static int item_addtolibrary_action_cb(void);
static int item_gotolibrary_action_cb(void);
static int item_importmatch_action_cb(void);
static int item_importmatchbybatch_action_cb(void);
static int item_matchlibrary_action_cb(void);
static int item_searchengine_action_cb(void);
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
static int toggle_editmode_cb();
static int toggle_cmdmode_cb();
void error_callback(void);
static int j_cb(Ihandle*, int);
static int k_cb(Ihandle*, int);
static int G_cb(Ihandle*, int);
static int editmode_letter_cb(Ihandle*, int);
static int editmode_digit_cb(Ihandle*, int);
static int minus_cb(Ihandle*, int);
static int bracketleft_cb(Ihandle*, int);
static int bracketright_cb(Ihandle*, int);
static int backspace_cb(Ihandle*, int);
static int space_cb(Ihandle*, int);
static int cr_cb(Ihandle*, int);
static int esc_cb(Ihandle*, int);
static int left_cb(Ihandle*, int);
static int right_cb(Ihandle*, int);
static int pgup_cb(Ihandle*, int);
static int pgdn_cb(Ihandle*, int);
static int update_sb_mode(void);
static int update_sb_msg(const char*);
static int update_sb_lib();
static int goto_first_position_cb(void);
static int goto_prev_position_cb(void);
static int goto_next_position_cb(void);
static int goto_last_position_cb(void);
static int goto_position_cb(int*);
static int board_direction_left_cb(void);
static int board_direction_right_cb(void);
static int checker_analysis_action_cb(Ihandle*);
static int display_player_on_roll_up(void);
static int display_player_on_roll_down(void);
// END Prototypes



/************************** Data *************************/

/* BEGIN Data */

#define PLAYER1 1
#define PLAYER2 -1
#define PLAYER1_POINTLABEL "*abcdefghijklmnopqrstuvwxyz"
#define PLAYER2_POINTLABEL "YABCDEFGHIJKLMNOPQRSTUVWX*Z"
#define POSITION_NUMBER_MAX 10000
#define CHECKER_ANALYSIS_NUMBER_MAX 100
#define LINE_NBMAX 1000
#define LINE_LENGTHMAX 256

enum txtfiletype { TXT_POSITION, TXT_MATCH, TXT_UNKNOWN };
typedef enum txtfiletype txtfiletype_t;

char hash[1000];

typedef struct
{
    int checker[26];
    int cube;
    int p1_score; // 2=2-away; 1=crawford; 0=postcrawford; -1=unlimited;
    int p2_score;
    int dice[2];
    int cube_action; //1=yes 0=no (hence roll action)
    int player_on_roll; //PLAYER1 or PLAYER2
} POSITION;

typedef struct
{
    char depth[20];
    char move1[10];
    char move2[10];
    char move3[10];
    char move4[10];
    double equity;
    double error;
    double p1_w;
    double p1_g;
    double p1_b;
    double p2_w;
    double p2_g;
    double p2_b;
} CHECKER_ANALYSIS;

typedef struct
{
    char depth[20];
    double p1_w;
    double p1_g;
    double p1_b;
    double p2_w;
    double p2_g;
    double p2_b;
    double cubeless_equity_nd;
    double cubeless_equity_d;
    double cubeful_equity_nd; //no double
    double cubeful_equity_dt; //double take
    double cubeful_equity_dp; //double pass
    double error_nd;
    double error_dt;
    double error_dp;
    int best_cube_action; // 0=nd 1=dt 2=dp 3=tgp 4=tgt
    double percentage_wrong_pass_make_good_double;
    int beaver;
} CUBE_ANALYSIS;

typedef struct
{
    char xgid[1000];
    char p1_name[100];
    char p2_name[100];
    char comment[10000];
    char misc[10000];
} METADATA;

typedef struct
{
    char site[30];
    char match_id[30];
    char event[200];
    char player1_name[100];
    char player2_name[100];
    char eventDate[20];
    char eventTime[20];
    char clockType[20];
    int matchLength;
    int player1_score;
    int player2_score;
} GAME;

typedef struct
{
    int dice[2];
    char move1[10];
    char move2[10];
    char move3[10];
    char move4[10];
    int player_on_roll; //PLAYER1 or PLAYER2
} MOVE;

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
    .dice = {3, 1},
    .cube_action = 0,
    .player_on_roll = PLAYER1,
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
    .dice = {3, 1},
    .cube_action = 0,
    .player_on_roll = PLAYER1,
};

const METADATA M_VOID = {
    .xgid = {'\0'},
    .p1_name = {'\0'},
    .p2_name = {'\0'},
    .comment = {'\0'},
    .misc = {'\0'},
};

const CHECKER_ANALYSIS CA_VOID = {
    .depth={'\0'},
    .move1={'\0'},
    .move2={'\0'},
    .move3={'\0'},
    .move4={'\0'},
    .equity=0,
    .error=0,
    .p1_w=0,
    .p1_g=0,
    .p1_b=0,
    .p2_w=0,
    .p2_g=0,
    .p2_b=0,
};

const CUBE_ANALYSIS D_VOID = {
    .depth={'\0'},
    .p1_w=0,
    .p1_g=0,
    .p1_b=0,
    .p2_w=0,
    .p2_g=0,
    .p2_b=0,
    .cubeless_equity_nd=0,
    .cubeless_equity_d=0,
    .cubeful_equity_nd=0,
    .cubeful_equity_dt=0,
    .cubeful_equity_dp=0,
    .error_nd=0,
    .error_dt=0,
    .error_dp=0,
    .best_cube_action=0,
    .percentage_wrong_pass_make_good_double=0,
};

const GAME G_VOID = {
    .site={'\0'},
    .match_id={'\0'},
    .event={'\0'},
    .player1_name={'\0'},
    .player2_name={'\0'},
    .eventDate={'\0'},
    .eventTime={'\0'},
    .clockType={'\0'},
    .matchLength=0,
    .player1_score=0,
    .player2_score=0,
};

const MOVE MV_VOID = {
    .dice={0,0},
    .move1={'\0'},
    .move2={'\0'},
    .move3={'\0'},
    .move4={'\0'},
    .player_on_roll=0,
};



POSITION pos;
POSITION *pos_ptr, *pos_prev_ptr, *pos_next_ptr;
bool is_pointletter_active = false;

POSITION pos_buffer;
POSITION pos_list[POSITION_NUMBER_MAX],
         pos_list_tmp[POSITION_NUMBER_MAX];
int pos_list_id[POSITION_NUMBER_MAX],
    pos_list_id_tmp[POSITION_NUMBER_MAX];
int pos_nb, pos_index;

CHECKER_ANALYSIS ca[CHECKER_ANALYSIS_NUMBER_MAX]; //checker analysis
CHECKER_ANALYSIS *ca_ptr;
int ca_nb; int ca_index=0;

CUBE_ANALYSIS da; //double analysis
CUBE_ANALYSIS *da_ptr;
int da_nb; int da_index=0;

int find_index_from_int(int v, int* a, int nb){
    for(int i=0;i<nb;i++){
        if(a[i]==v) return i;
    }
    return 0;
}

int char_in_string(const char c, const char* s) {
    int index;
    char *e;
    e = strchr(s, c);
    index = (int) (e - s);
    return index;
}

void copy_position(const POSITION* a, POSITION* b){
    for(int i=0;i<26;i++) b->checker[i]=a->checker[i];
    b->cube=a->cube;
    b->p1_score=a->p1_score;
    b->p2_score=a->p2_score;
    b->dice[0]=a->dice[0];
    b->dice[1]=a->dice[1];
    b->cube_action=a->cube_action;
    b->player_on_roll=a->player_on_roll;
    for(int i=0;i<2;i++) b->dice[i]=a->dice[i];
    b->cube_action=a->cube_action;
}

void position_print(const POSITION* p) {
    printf("\nposition_print\n");
    printf("checker:\n");
    for(int i=0; i<26; i++)
    {
        printf("%i: %i\n", i, p->checker[i]);
    }
    printf("cube: %i\n", p->cube);
    printf("p1_score: %i\n", p->p1_score);
    printf("p2_score: %i\n", p->p2_score);
    printf("dice: %i, %i\n", p->dice[0], p->dice[1]);
    printf("cube_action: %i\n", p->cube_action);
    printf("player_on_roll: %i\n", p->player_on_roll);
}

void cube_analysis_print(const CUBE_ANALYSIS* d){
    printf("\ncube_analysis_print\n");
    printf("depth: %s\n",d->depth);
    printf("p1_w: %f\n",d->p1_w);
    printf("p1_g: %f\n",d->p1_g);
    printf("p1_b: %f\n",d->p1_b);
    printf("p2_w: %f\n",d->p2_w);
    printf("p2_g: %f\n",d->p2_g);
    printf("p2_b: %f\n",d->p2_b);
    printf("cubeless_equity_nd: %f\n",d->cubeless_equity_nd);
    printf("cubeless_equity_d: %f\n",d->cubeless_equity_d);
    printf("cubeful_equity_nd: %f\n",d->cubeful_equity_nd);
    printf("cubeful_equity_dt: %f\n",d->cubeful_equity_dt);
    printf("cubeful_equity_dp: %f\n",d->cubeful_equity_dp);
    printf("error_nd: %f\n",d->error_nd);
    printf("error_dt: %f\n",d->error_dt);
    printf("error_dp: %f\n",d->error_dp);
    printf("best_cube_action: %i\n",d->best_cube_action);
    printf("percentage_wrong_pass_make_good_double: %f\n",d->percentage_wrong_pass_make_good_double);
}

void checker_analysis_print(const CHECKER_ANALYSIS* c) {
    printf("\nchecker_analysis_print\n");
    printf("depth:%s\n",c->depth);
    printf("move1:%s\n",c->move1);
    printf("move2:%s\n",c->move2);
    printf("move3:%s\n",c->move3);
    printf("move4:%s\n",c->move4);
    printf("equity:%f\n",c->equity);
    printf("error:%f\n",c->error);
    printf("p1_w:%f\n",c->p1_w);
    printf("p1_g:%f\n",c->p1_g);
    printf("p1_b:%f\n",c->p1_b);
    printf("p2_w:%f\n",c->p2_w);
    printf("p2_g:%f\n",c->p2_g);
    printf("p2_b:%f\n",c->p2_b);
}

void metadata_print(const METADATA* m) {
    printf("\nmetadata_print\n");
    printf("xgid:%s\n",m->xgid);
    printf("p1_name:%s\n",m->p1_name);
    printf("p2_name:%s\n",m->p2_name);
    printf("comment:%s\n",m->comment);
    printf("misc:%s\n",m->misc);
}

void game_print(const GAME* m) {
    printf("\ngame_print\n");
    printf("site:%s\n",m->site);
    printf("match_id:%s\n",m->match_id);
    printf("event:%s\n",m->event);
    printf("eventDate:%s\n",m->eventDate);
    printf("eventTime:%s\n",m->eventTime);
    printf("clockType:%s\n",m->clockType);
    printf("player1_name:%s\n",m->player1_name);
    printf("player2_name:%s\n",m->player2_name);
    printf("matchLength:%i\n",m->matchLength);
    printf("player1_score:%i\n",m->player1_score);
    printf("player2_score:%i\n",m->player2_score);
}

void move_print(const MOVE* m) {
    printf("\nmove_print\n");
    printf("dice:%i %i\n",m->dice[0], m->dice[1]);
    printf("move1:%s\n",m->move1);
    printf("move2:%s\n",m->move2);
    printf("move3:%s\n",m->move3);
    printf("move4:%s\n",m->move4);
    printf("player_on_roll:%i\n",m->player_on_roll);
}


void int_swap(int* i, int* j) {
    int t; t = *i; *i = *j; *j = t;
}

int convert_charp_to_array(const char *c, char *c_array, const int n_array){
    int n=strlen(c);
    if(n_array<=n) {
        printf("ERR: array no big enough for string conversion.\n");
        return 0;
    }
    for(int i=0; i<=n; i++){
        c_array[i]=c[i];
    }
    c_array[n+1]='\0';
    return 1;
}

char* pos_to_str(const POSITION* p)
{
    const char p1[27] = PLAYER1_POINTLABEL;
    const char p2[27] = PLAYER2_POINTLABEL;
    char p1_score[10];
    char p2_score[10];
    char _d[2];
    char* c = malloc(100 * sizeof(char));
    memcpy(c, "\0", 1);
    sprintf(p1_score, "%d", p->p1_score);
    sprintf(p2_score, "%d", p->p2_score);
    printf("p1_score: %s\np2_score: %s\n", p1_score, p2_score);
    strcat(c, p1_score);
    strcat(c, ",");
    strcat(c, p2_score);
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
    const char p1[27] = PLAYER1_POINTLABEL;
    const char p2[27] = PLAYER2_POINTLABEL;
    char p1_score[10];
    char p2_score[10];
    char _d[2];
    char* c = malloc(100 * sizeof(char));
    char* c_spare = malloc(50 * sizeof(char));
    char* c_point = malloc(50 * sizeof(char));
    memcpy(c, "\0", 1);
    memcpy(c_spare, "\0", 1);
    memcpy(c_point, "\0", 1);
    sprintf(p1_score, "%d", p->p1_score);
    sprintf(p2_score, "%d", p->p2_score);
    strcat(c, p1_score);
    strcat(c, ",");
    strcat(c, p2_score);
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

void get_prev_position(){
    if(pos_index==0) return;
    pos_index-=1;
    pos_ptr=&pos_list[pos_index];
}

void get_next_position(){
    if(pos_index==pos_nb-1) return;
    pos_index+=1;
    pos_ptr=&pos_list[pos_index];
}

void get_first_position(){
    pos_index=0;
    pos_ptr=&pos_list[pos_index];
}

void get_last_position(){
    pos_index=pos_nb-1;
    pos_ptr=&pos_list[pos_index];
}

int get_position(int* id){
    pos_index=find_index_from_int(*id, pos_list_id,
            POSITION_NUMBER_MAX );
    pos_ptr=&pos_list[pos_index];
    return 1;
}

void filter_position_by_checkeroff(int omin){
    printf("\nfilter_position_by_checkeroff\n");
    int off1, off2;
    int j=0;
    for(int i=0;i<pos_nb;i++){
        compute_checkeroff(&pos_list[i], &off1, &off2);
        if((off1>=omin) || (off2>=omin)){
            printf("j %i\n",j);
            printf("i omin off1 off2: %i %i %i %i\n", i,omin,off1,off2);
            pos_list_tmp[j]=pos_list[i];
            pos_list_id_tmp[j]=pos_list_id[i];
            j+=1;
        }
    }
    for(int i=0;i<j;i++){
            pos_list[i]=pos_list_tmp[i];
            pos_list_id[i]=pos_list_id_tmp[i];
    }
    pos_nb=j;
    printf("pos_nb: %i\n", pos_nb);
}

void filter_position_by_pipcount(int pmin, int pmax, bool is_absdiff){
    printf("\nfilter_position_by_pipcount\n");
    int pip1, pip2, diff;
    int j=0;
    for(int i=0;i<pos_nb;i++){
        compute_pipcount(&pos_list[i], &pip1, &pip2);
        if(is_absdiff) { diff=pip1-pip2;
        } else { diff=abs(pip1-pip2); }
        if((diff>=pmin) && (diff<=pmax)){
            printf("j %i\n",j);
            printf("i diff pmin pmax: %i %i %i %i\n", i,diff,pmin,pmax);
            pos_list_tmp[j]=pos_list[i];
            pos_list_id_tmp[j]=pos_list_id[i];
            j+=1;
        }
    }
    for(int i=0;i<j;i++){
            pos_list[i]=pos_list_tmp[i];
            pos_list_id[i]=pos_list_id_tmp[i];
    }
    pos_nb=j;
    printf("pos_nb: %i\n", pos_nb);
}

void filter_position_by_backchecker(int bc_player, int bc_num){
    printf("\nfilter_position_by_backchecker\n");
    int j=0;
    for(int i=0;i<pos_nb;i++){
        int _n=0;
        if(bc_player==PLAYER1){
            for(int k=14;k<26;k++){
                if(pos_list[i].checker[k]>0){
                    _n+=pos_list[i].checker[k];
                }
            }
        } else if(bc_player==PLAYER2){
            for(int k=0;k<12;k++){
                if(pos_list[i].checker[k]<0){
                    _n+=abs(pos_list[i].checker[k]);
                }
            }
        }
        if(_n==bc_num){
            pos_list_tmp[j]=pos_list[i];
            pos_list_id_tmp[j]=pos_list_id[i];
            j+=1;
            printf("i pos pos_id: %i %i %i\n", i, pos_list[i], pos_list_id[i]);
        }
    }
    for(int i=0;i<j;i++){
        pos_list[i]=pos_list_tmp[i];
        pos_list_id[i]=pos_list_id_tmp[i];
    }
    pos_nb=j;
    printf("pos_nb: %i\n", pos_nb);
}

void filter_position_by_checker_in_the_zone(int z_player, int z_num){
    printf("\nfilter_position_by_checker_in_the_zone\n");
    int j=0;
    for(int i=0;i<pos_nb;i++){
        int _n=0;
        if(z_player==PLAYER1){
            for(int k=1;k<=12;k++){
                if(pos_list[i].checker[k]>0){
                    _n+=pos_list[i].checker[k];
                }
            }
        } else if(z_player==PLAYER2){
            for(int k=13;k<=24;k++){
                if(pos_list[i].checker[k]<0){
                    _n+=abs(pos_list[i].checker[k]);
                }
            }
        }
        if(_n==z_num){
            pos_list_tmp[j]=pos_list[i];
            pos_list_id_tmp[j]=pos_list_id[i];
            j+=1;
            printf("i pos pos_id: %i %i %i\n", i, pos_list[i], pos_list_id[i]);
        }
    }
    for(int i=0;i<j;i++){
        pos_list[i]=pos_list_tmp[i];
        pos_list_id[i]=pos_list_id_tmp[i];
    }
    pos_nb=j;
    printf("pos_nb: %i\n", pos_nb);
}

FILE *open_input(const char *filename){
    FILE *f;
    errno=0;
    if(filename==NULL) filename='\0';
    f=fopen(filename,"r");
    if(f==NULL)
        fprintf(stderr,
                "open_input(\"%s\") failed: %s\n",
                filename,strerror(errno));
    return f;
}

txtfiletype_t check_if_file_is_position_or_match(FILE *f){
    printf("\ncheck_if_file_is_position_or_match\n");
    char lines[LINE_NBMAX][LINE_LENGTHMAX]; int nb;
    nb=0;
    lines[nb][0]='\0';
    while(fgets(lines[nb],sizeof(lines[nb]),f)){
        lines[nb][strcspn(lines[nb],"\r\n")]=0; //delete \n end of line
        nb+=1;
        lines[nb][0]='\0';
    }
    for(int i=0;i<nb;i++){
        printf("l:%s\n",lines[i]);
        if(strstr(lines[i],"XGID=")!=0) return TXT_POSITION; 
        if(strstr(lines[i],"point match")!=0) return TXT_MATCH;
        if(strstr(lines[i],"Game")!=0) return TXT_MATCH;
    }
    return TXT_UNKNOWN;
}

int close_file(FILE *f){
    int s=0;
    if(f==NULL) return 0;
    errno=0;
    s=fclose(f);
    if(s==EOF) perror("Close failed");
    return s;
}

static int invert_position(POSITION* p){
    printf("\ninvert_position\n");
    copy_position(p, &pos_buffer);
    for(int i=0;i<26;i++){
        pos_buffer.checker[i]=-p->checker[25-i];
    }
    pos_buffer.cube*=-1;
    pos_buffer.player_on_roll*=-1;
    pos_buffer.p1_score=p->p2_score;
    pos_buffer.p2_score=p->p1_score;
    copy_position(&pos_buffer, p);
}

void convert_xgid_to_position(const char *l, POSITION *p){
    char t[100]; char* token[10]; int i;
    const char *f1="-ABCDEFGHIKLMNOPQRSTUVWXYZ";
    const char *f2="-abcdefghiklmnopqrstuvwxyz";
    l+5; strncpy(t,l,55);
    char *c = strtok(t, ":");
    i=0; while(c!=NULL){
        token[i]=c; i+=1;
        c=strtok(NULL, ":");
    }
    int p1_sign=atoi(token[3]); //in theory, -1 means opponent.
                                //but in blunderDB, p1 is player downside.
                                //this is important for match import
    char _checker[27];
    strcpy(_checker,token[0]);
    for(int i=0;i<26;i++){
        int k;
        if(p1_sign==1) k=i;
        if(p1_sign==-1) k=25-i;
        if(_checker[i]=='-'){
            p->checker[k]=0;
        } else if(isupper(_checker[i])){
            int n=char_in_string(_checker[i],f1);
            p->checker[k]=n*p1_sign;
        } else if(islower(_checker[i])){
            int n=char_in_string(_checker[i],f2);
            p->checker[k]=-n*p1_sign;
        }
    }
    int cube_value=atoi(token[1]);
    int cube_owner=atoi(token[2])*p1_sign; //0 middle 1=p1 -1=p2
    p->cube=cube_value*cube_owner;
    int roll=atoi(token[4]);
    if(roll==0) p->cube_action=1;
    p->dice[0]=roll/10;
    p->dice[1]=(int) fmod((double)roll,10.); // -lm?
    int p1_score=atoi(token[5]);
    int p2_score=atoi(token[6]);
    if(p1_sign==-1) int_swap(&p1_score,&p2_score);
    int is_crawford=atoi(token[7]);
    int match_length=atoi(token[8]);
    p->p1_score=match_length-p1_score;
    p->p2_score=match_length-p2_score;
    if(p1_sign<0) invert_position(p);
}

void convert_position_to_xgid(const POSITION *p, char *l){
    printf("\nconvert_position_to_xgid\n");
    char t[100];t[0]='\0';
    const char *f1="-ABCDEFGHIKLMNOPQRSTUVWXYZ";
    const char *f2="-abcdefghiklmnopqrstuvwxyz";
    strcat(l, "XGID=");
    for(int i=0;i<26;i++){
        int k=p->checker[i];
        if(k>0){
            sprintf(t,"%c",f1[k]);
            strcat(l,t);
        }else{
            sprintf(t,"%c",f2[-k]);
            strcat(l,t);
        }
    }
    strcat(l,":");
    sprintf(t,"%d",abs(p->cube));
    strcat(l,t);
    strcat(l,":");
    if(p->cube==0){
        strcat(l,"0");
    }else if(p->cube>0){
            strcat(l,"1");
    }else if(p->cube<0){
        strcat(l,"-1");
    }
    strcat(l,":");
    sprintf(t,"%d",p->player_on_roll);
    strcat(l,t);
    strcat(l,":");
    sprintf(t,"%d",p->dice[0]*10+p->dice[1]);
    strcat(l,t);
    strcat(l,":");
    int match_length=(int) fmax((double)p->p1_score,(double)p->p2_score);
    printf("match length %i\n",match_length);
    if(p->p1_score==0){
        sprintf(t,"%d",match_length-1);
    }else if(p->p1_score==-1){
        sprintf(t,"%d",0);
    }else {
        sprintf(t,"%d",match_length-(p->p1_score));
    }
    strcat(l,t);
    strcat(l,":");
    if(p->p2_score==0){
        sprintf(t,"%d",match_length-1);
    }else if(p->p2_score==-1){
        sprintf(t,"%d",0);
    }else{
        sprintf(t,"%d", match_length-(p->p2_score));
    }
    strcat(l,t);
    strcat(l,":");
    if(p->p1_score<0||p->p2_score<0){
        strcat(l,"3:0");
    }else if(p->p1_score==1||p->p2_score==1){
        strcat(l,"1");
        strcat(l,":");
        sprintf(t,"%d",match_length);
        strcat(l,t);
    } else {
        strcat(l,"0");
        strcat(l,":");
        sprintf(t,"%d",match_length);
        strcat(l,t);
    }
    strcat(l,":10");
    printf("xgid: %s\n",l);
}


bool has_extension(const char *filename, const char *extension) {
    const char *dot = strrchr(filename, '.');
    if (!dot || dot == filename) {
        return false; // No extension found
    }
    return strcmp(dot + 1, extension) == 0;
}


/* END Data */


/************************ Database ***********************/
/* BEGIN Database */
#define LIBRARIES_NUMBER_MAX 1000
#define LIBRARY_NAME_MAX 50

sqlite3 *db = NULL;
sqlite3_stmt *stmt;
bool is_db_saved = true;
int rc;
char *errMsg = 0;
char db_file[10240];
int parse_from_file = 0;

const char *sql_library =
"CREATE TABLE library ("
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
"player1_score INTEGER,"
"player2_score INTEGER,"
"dice1 INTEGER,"
"dice2 INTEGER,"
"cube_position INTEGER,"
"player_on_roll INTEGER,"
"cube_action INTEGER,"
"hash TEXT,"
"comment TEXT"
");";

const char* sql_catalog =
"CREATE TABLE catalog ("
"position_id INTEGER,"
"library_id INTEGER,"
"FOREIGN KEY(position_id) REFERENCES position(id),"
"FOREIGN KEY(library_id) REFERENCES library(id)"
");";

const char* sql_metadata =
"CREATE TABLE metadata ("
"position_id INTEGER,"
"xgid TEXT,"
"player1_name TEXT,"
"player2_name TEXT,"
"comment TEXT,"
"misc TEXT,"
"FOREIGN KEY(position_id) REFERENCES position(id)"
");";

const char* sql_checker_analysis =
"CREATE TABLE checker_analysis ("
"position_id INTEGER,"
"depth TEXT,"
"move1 TEXT,"
"move2 TEXT,"
"move3 TEXT,"
"move4 TEXT,"
"equity REAL,"
"error REAL,"
"p1_w REAL,"
"p1_g REAL,"
"p1_b REAL,"
"p2_w REAL,"
"p2_g REAL,"
"p2_b REAL,"
"FOREIGN KEY(position_id) REFERENCES position(id)"
");";

const char* sql_cube_analysis =
"CREATE TABLE cube_analysis ("
"position_id INTEGER,"
"depth TEXT,"
"p1_w REAL,"
"p1_g REAL,"
"p1_b REAL,"
"p2_w REAL,"
"p2_g REAL,"
"p2_b REAL,"
"cubeless_equity_nd REAL,"
"cubeless_equity_d REAL,"
"cubeful_equity_nd REAL,"
"cubeful_equity_dt REAL,"
"cubeful_equity_dp REAL,"
"error_nd REAL,"
"error_dt REAL,"
"error_dp REAL,"
"best_cube_action INTEGER,"
"percentage_wrong_pass_make_good_double REAL,"
"beaver INTEGER,"
"FOREIGN KEY(position_id) REFERENCES position(id)"
");";

const char *sql_game =
"CREATE TABLE game ("
"id INTEGER PRIMARY KEY AUTOINCREMENT,"
"site TEXT,"
"match_id TEXT,"
"event TEXT,"
"player1_name TEXT,"
"player2_name TEXT,"
"eventDate TEXT,"
"eventTime TEXT,"
"clockType TEXT,"
"matchLength INTEGER,"
"player1_score INTEGER,"
"player2_score INTEGER,"
"game_prev INTEGER," //begin of match->game_prev empty
"game_next INTEGER," //end of match-<game_next empty
"FOREIGN KEY(game_prev) REFERENCES game(id),"
"FOREIGN KEY(game_next) REFERENCES game(id)"
");";

const char* sql_move =
"CREATE TABLE move ("
"id INTEGER PRIMARY KEY AUTOINCREMENT,"
"game_id INTEGER,"
"dice1 INTEGER,"
"dice2 INTEGER,"
"move1 TEXT,"
"move2 TEXT,"
"move3 TEXT,"
"move4 TEXT,"
"player_on_roll INTEGER,"
"move_prev INTEGER," //begin of game->move_prev empty
"move_next INTEGER," //end of game->move_next empty
"position_id INTEGER,"
"FOREIGN KEY(game_id) REFERENCES game(id),"
"FOREIGN KEY(move_prev) REFERENCES move(id),"
"FOREIGN KEY(move_next) REFERENCES move(id),"
"FOREIGN KEY(position_id) REFERENCES position(id)"
");";




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

    printf("Try to create position table.\n");
    execute_sql(db, sql_position);

    printf("Try to create catalog table.\n");
    execute_sql(db, sql_catalog);

    printf("Try to create library table.\n");
    execute_sql(db, sql_library);

    printf("Try to create metadata table.\n");
    execute_sql(db, sql_metadata);

    printf("Try to create checker_analysis table.\n");
    execute_sql(db, sql_checker_analysis);

    printf("Try to create cube_analysis table.\n");
    execute_sql(db, sql_cube_analysis);

    printf("Try to create game table.\n");
    execute_sql(db, sql_game);

    printf("Try to create move table.\n");
    execute_sql(db, sql_move);

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
        printf("Can't close database. Maybe already closed. ERR: %s\n", sqlite3_errmsg(db));
    } else {
        printf("Closed database successfully\n");
    }
    return rc;
}

int db_last_insert_id(sqlite3 *db, char *t){
    printf("\ndb_last_insert_id\n");
    char sql[1000]; sql[0]='\0';
    sprintf(sql,"SELECT seq FROM sqlite_sequence WHERE name=\"%s\";",t);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    int id, i;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        id=sqlite3_column_int(stmt,0);
        i+=1;
    }
    sqlite3_finalize(stmt);
    return id;



}

int db_insert_position(sqlite3 *db, const POSITION *p){
    printf("\ndb_insert_position\n");
    char _s[4]; char *h;
    char sql_add_position[1000];
    h=pos_to_str(p);
    convert_charp_to_array(h, hash, 1000);
    printf("hash: %s\n", hash);
    sql_add_position[0]='\0';
    strcat(sql_add_position, "INSERT INTO position ");
    strcat(sql_add_position, "(p0, p1, p2, p3, p4, p5, ");
    strcat(sql_add_position, "p6, p7, p8, p9, p10, p11, ");
    strcat(sql_add_position, "p12, p13, p14, p15, p16, p17, ");
    strcat(sql_add_position, "p18, p19, p20, p21, p22, p23, ");
    strcat(sql_add_position, "p24, p25, ");
    strcat(sql_add_position, "player1_score, player2_score, ");
    strcat(sql_add_position, "dice1, dice2, ");
    strcat(sql_add_position, "cube_position, player_on_roll, cube_action, ");
    strcat(sql_add_position, "hash) ");
    strcat(sql_add_position, "VALUES ");
    strcat(sql_add_position, "(");
    for(int i=0;i<26;i++){
        sprintf(_s, "%d", p->checker[i]);
        strcat(sql_add_position, _s);
        strcat(sql_add_position, ", ");
    }
    sprintf(_s, "%d, %d, %d, %d, %d, %d, %d",
            p->p1_score, p->p2_score, p->dice[0], p->dice[1],
            p->cube, p->player_on_roll, p->cube_action);
    strcat(sql_add_position, _s);
    strcat(sql_add_position, ", \"");
    strcat(sql_add_position, hash);
    strcat(sql_add_position, "\");");
    printf("sql insert: %s\n", sql_add_position);
    printf("Try to add new position.\n");
    execute_sql(db, sql_add_position); 
    return 1;
}

int db_insert_metadata(sqlite3 *db, const int *pid, const METADATA *m){
    printf("\ndb_insert_metadata\n");
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0';
    strcat(sql,"INSERT INTO metadata ");
    strcat(sql,"(position_id,xgid,player1_name,player2_name,comment,misc) ");
    strcat(sql,"VALUES (");
    sprintf(t,"%d,\"%s\",\"%s\",\"%s\",\"%s\",\"%s\");",
            *pid,m->xgid,m->p1_name,m->p2_name,m->comment,m->misc);
    strcat(sql,t);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}

int db_delete_if_exist_metadata(sqlite3* db, int pid){
    printf("\ndb_delete_if_exist_metadata\n");
    char sql[10000]; sql[0]='\0';
    sprintf(sql,"DELETE FROM metadata WHERE position_id=%d;",pid);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}

int db_insert_checker_analysis(sqlite3 *db, const int *pid,
        const CHECKER_ANALYSIS *a){
    printf("\ndb_insert_checker_analysis\n");
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0';
    strcat(sql,"INSERT INTO checker_analysis ");
    strcat(sql,"(position_id,depth,move1,move2,move3,move4,");
    strcat(sql,"equity,error,p1_w,p1_g,p1_b,p2_w,p2_g,p2_b) ");
    strcat(sql,"VALUES (");
    sprintf(t,"%d,\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",%f,%f,%f,%f,%f,%f,%f,%f);",
            *pid,a->depth,a->move1,a->move2,a->move3,a->move4,a->equity,a->error,
            a->p1_w,a->p1_g,a->p1_b,a->p2_w,a->p2_g,a->p2_b);
    strcat(sql,t);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}

int db_delete_if_exist_checker_analysis(sqlite3* db, int pid){
    printf("\ndb_delete_if_exist_checker_analysis\n");
    char sql[10000]; sql[0]='\0';
    sprintf(sql,"DELETE FROM checker_analysis WHERE position_id=%d;",pid);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}


int db_insert_cube_analysis(sqlite3 *db, const int *pid,
        const CUBE_ANALYSIS *a){
    printf("\ndb_insert_cube_analysis\n");
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0';
    strcat(sql,"INSERT INTO cube_analysis ");
    strcat(sql,"(position_id,depth,p1_w,p1_g,p1_b,p2_w,p2_g,p2_b,");
    strcat(sql,"cubeless_equity_nd,cubeless_equity_d,");
    strcat(sql,"cubeful_equity_nd,cubeful_equity_dt,cubeful_equity_dp,");
    strcat(sql,"error_nd,error_dt,error_dp,");
    strcat(sql,"best_cube_action,percentage_wrong_pass_make_good_double,beaver) ");
    strcat(sql,"VALUES (");
    sprintf(t,"%d,\"%s\",%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%d,%f,%d);",
            *pid,a->depth,a->p1_w,a->p1_g,a->p1_b,a->p2_w,a->p2_g,a->p2_b,
            a->cubeless_equity_nd,a->cubeless_equity_d,
            a->cubeful_equity_nd,a->cubeful_equity_dt,a->cubeful_equity_dp,
            a->error_nd,a->error_dt,a->error_dp,
            a->best_cube_action,a->percentage_wrong_pass_make_good_double,
            a->beaver);
    strcat(sql,t);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}

int db_delete_if_exist_cube_analysis(sqlite3* db, int pid){
    printf("\ndb_delete_if_exist_cube_analysis\n");
    char sql[10000]; sql[0]='\0';
    sprintf(sql,"DELETE FROM cube_analysis WHERE position_id=%d;",pid);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}


int db_update_position(sqlite3* db, int* id, const POSITION* p){
    char sql[10000]; char *h;
    char t[10000];
    sql[0]='\0'; t[0]='\0';
    h=pos_to_str(p);
    convert_charp_to_array(h, hash, 1000);
    strcat(sql, "UPDATE position SET ");
    for(int i=0; i<26; i++){
        sprintf(t, "p%i = %i, ", i, p->checker[i]);
        strcat(sql, t);
    }
    sprintf(t, "player1_score = %d, ", p->p1_score);
    strcat(sql, t);
    sprintf(t, "player2_score = %d, ", p->p2_score);
    strcat(sql, t);
    sprintf(t, "cube_position = %d,  ", p->cube);
    strcat(sql, t);
    strcat(sql, "hash = \"");
    strcat(sql, hash);
    strcat(sql, "\" ");
    sprintf(t, "WHERE id = %d;", *id);
    strcat(sql, t);
    printf("sql: %s\n", sql);
    execute_sql(db, sql); 
    return 1;
}

int db_select_position(sqlite3* db, int* pos_nb,
        int* pos_list_id, POSITION* pos_list){
    printf("\ndb_select_position\n");
    const char *sql = "SELECT * FROM position;";
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }

    *pos_nb=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        pos_list_id[*pos_nb]=sqlite3_column_int(stmt,0);
        for(int i=0;i<26;i++){
            pos_list[*pos_nb].checker[i]=sqlite3_column_int(stmt,i+1);
        }
        pos_list[*pos_nb].p1_score=sqlite3_column_int(stmt,27);
        pos_list[*pos_nb].p2_score=sqlite3_column_int(stmt,28);
        pos_list[*pos_nb].dice[0]=sqlite3_column_int(stmt,29);
        pos_list[*pos_nb].dice[1]=sqlite3_column_int(stmt,30);
        pos_list[*pos_nb].cube=sqlite3_column_int(stmt,31);
        pos_list[*pos_nb].player_on_roll=sqlite3_column_int(stmt,32);
        pos_list[*pos_nb].cube_action=sqlite3_column_int(stmt,33);
        const char *hash=sqlite3_column_text(stmt,34);
        *pos_nb+=1;
    }
    if(rc!=SQLITE_DONE){
        printf("Failed to execute statement: %s\n",
                sqlite3_errmsg(db));
    }
    sqlite3_finalize(stmt);
    return 1;
}

int db_get_library_id_from_name(sqlite3* db, const char *name,
        int *id){
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0';
    printf("\ndb_get_library_id_from_name\n");
    /* strcat(sql, "SELECT id FROM library WHERE */
    sprintf(sql, "SELECT id FROM library WHERE name = \"%s\";",
            name);
    printf("sql: %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        *id=sqlite3_column_int(stmt,0);
    }
    return 1;
}



bool db_library_exists(sqlite3* db, const char *l){
    char sql[10000], t[10000]; 
    sql[0]='\0'; t[0]='\0';
    strcat(sql, "SELECT id,name FROM library WHERE ");
    sprintf(t, "name = \"%s\";", l);
    strcat(sql, t);
    printf("sql: %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }
    int id;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        id=sqlite3_column_int(stmt,0);
        return true;
    }
    return false;
}

bool db_is_valid_library_name(const char *l){
    printf("\ndb_is_valid_library_name\n");
    int n=strlen(l);
    if(l[0]=='-') return false;
    for(int i=0;i<n;i++){
        if(!isalnum(l[i])
                && l[i]!='-'
                && l[i]!='_'){
            return false;
        }
    }
    return true;
}

int db_select_position_from_libraries(sqlite3* db, char** cmdtoken,
        int token_nb, int* pos_nb, int* pos_list_id, POSITION* pos_list){
    printf("\ndb_select_position_from_library\n");
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0';
    /* strcat(sql,"SELECT DISTINCT * FROM position p "); */
    strcat(sql,"SELECT DISTINCT p.* FROM position p ");
    strcat(sql,"INNER JOIN catalog c ON p.id=c.position_id ");
    strcat(sql,"WHERE 1=0 ");
    char *l;
    for(int k=1;k<token_nb;k++){
        l=cmdtoken[k]; int l_id;
        if(!db_is_valid_library_name(l)){
            continue;
        }
        if(db_library_exists(db,l)){
            db_get_library_id_from_name(db,l,&l_id);
            sprintf(t, "or c.library_id = %d ", l_id); 
            strcat(sql, t);
        }
    }
    strcat(sql,";");
    printf("sql %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }

    *pos_nb=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        pos_list_id[*pos_nb]=sqlite3_column_int(stmt,0);
        for(int i=0;i<26;i++){
            pos_list[*pos_nb].checker[i]=sqlite3_column_int(stmt,i+1);
        }
        pos_list[*pos_nb].p1_score=sqlite3_column_int(stmt,27);
        pos_list[*pos_nb].p2_score=sqlite3_column_int(stmt,28);
        pos_list[*pos_nb].dice[0]=sqlite3_column_int(stmt,29);
        pos_list[*pos_nb].dice[1]=sqlite3_column_int(stmt,30);
        pos_list[*pos_nb].cube=sqlite3_column_int(stmt,31);
        pos_list[*pos_nb].player_on_roll=sqlite3_column_int(stmt,32);
        pos_list[*pos_nb].cube_action=sqlite3_column_int(stmt,33);
        const char *hash=sqlite3_column_text(stmt,34);
        *pos_nb+=1;
    }
    if(rc!=SQLITE_DONE){
        printf("Failed to execute statement: %s\n",
                sqlite3_errmsg(db));
    }
    sqlite3_finalize(stmt);
    return 1;
}

int db_select_position_from_specific_library(sqlite3* db,
        const char* lname, int* pos_nb, int* pos_list_id, POSITION* pos_list){
    printf("\ndb_select_position_from_specific_library\n");
    char* cmdtoken[2];
    cmdtoken[0]=":e";
    cmdtoken[1]=(char*)lname;
    printf("toot\n");
    /* strcat(cmdtoken[1],lname); */
    int token_nb=2;
    db_select_position_from_libraries(db,cmdtoken,
        token_nb,pos_nb,pos_list_id,pos_list);
    return 1;
}

int db_select_specific_position(sqlite3* db, const POSITION* p,
        const int force_cube, const int force_score,
        const int criteria_blunder, const int bmin, const int bmax,
        const int criteria_equity, const int emin, const int emax,
        int* p_nb, int* p_list_id, POSITION* p_list){
    printf("\ndb_select_specific_position\n");
    // add constraints due to blunder, pipcount, checkeroff
    char sql[10000], t[10000];
    sql[0]='\0'; t[0]='\0';
    strcat(sql, "SELECT DISTINCT p.* FROM position p ");
    strcat(sql, "LEFT JOIN checker_analysis ca ON p.id=ca.position_id ");
    strcat(sql, "LEFT JOIN cube_analysis d ON p.id=d.position_id ");
    strcat(sql, "WHERE 1=1 ");
    for(int i=0;i<26;i++){
        if(p->checker[i]>0){
            strcat(sql, "and ");
            sprintf(t, "p%i >= %i ", i, p->checker[i]);
            strcat(sql, t);
        } else if(p->checker[i]<0){
            strcat(sql, "and ");
            sprintf(t, "p%i <= %i ", i, p->checker[i]);
            strcat(sql, t);
        }
    }
    printf("force_score: %i\nforce_cube: %i\n", force_score, force_cube);
    if(force_score){
        printf("\nforce_score\n");
        sprintf(t, "and player1_score = %i and player2_score = %i ",
                p->p1_score, p->p2_score);
        strcat(sql, t);
    }
    if(force_cube){
        printf("\nforce_cube\n");
        sprintf(t, "and cube_position = %i ", p->cube);
        strcat(sql, t);
    }
    if(criteria_blunder){
        printf("\ncriteria_blunder\n");
        printf("TO BE IMPLEMENTED\n");
        //impossible to filyer by blunder without
        //knowing which move has been played.
        //Can be implemented after match support.
        //Must be a join on multiple tables:
        //moves, cube_analysis, checker_analysis
    }
    if(criteria_equity){
        printf("\ncriteria_equity\n");
        //idem as criteria_blunder
        //must be applied to the move actually played.
        //modification ca.equity necessary
        strcat(sql, "and (");
        sprintf(t,"(ca.equity>=%lf and ca.equity<=%lf) ",
                ((double)emin)/1000., ((double)emax)/1000.);
        strcat(sql,t);
        sprintf(t,"or (d.cubeful_equity_nd>=%lf and d.cubeful_equity_nd<=%lf))",
                ((double)emin)/1000., ((double)emax)/1000.);
        strcat(sql,t);
    }
    strcat(sql, ";");
    printf("sql: %s\n", sql);

    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }

    *p_nb=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        p_list_id[*p_nb]=sqlite3_column_int(stmt,0);
        for(int i=0;i<26;i++){
            p_list[*p_nb].checker[i]=sqlite3_column_int(stmt,i+1);
        }
        p_list[*p_nb].p1_score=sqlite3_column_int(stmt,27);
        p_list[*p_nb].p2_score=sqlite3_column_int(stmt,28);
        p_list[*p_nb].dice[0]=sqlite3_column_int(stmt,29);
        p_list[*p_nb].dice[1]=sqlite3_column_int(stmt,30);
        p_list[*p_nb].cube=sqlite3_column_int(stmt,31);
        p_list[*p_nb].player_on_roll=sqlite3_column_int(stmt,32);
        p_list[*p_nb].cube_action=sqlite3_column_int(stmt,33);
        const char *hash=sqlite3_column_text(stmt,34);
        *p_nb+=1;
    }

    if(rc!=SQLITE_DONE){
        printf("Failed to execute statement: %s\n",
                sqlite3_errmsg(db));
    }
    sqlite3_finalize(stmt);
    return 1;
}

int db_delete_position(sqlite3* db, const int* id){
    printf("\ndb_delete_position\n");
    char sql[100];
    sql[0]='\0';
    sprintf(sql, "DELETE FROM position WHERE id = %d;", *id);
    execute_sql(db, sql);
    db_select_position(db, &pos_nb,
            pos_list_id, pos_list);
    goto_last_position_cb();
    return 1;
}

int db_find_identical_position(sqlite3* db, const POSITION* p, bool* exist, int* nb, int* id)
{
    printf("\ndb_find_identical_position\n");
    char sql[10000]; char *h;
    char t[10000];
    sql[0]='\0'; t[0]='\0';
    h=pos_to_str(p);
    convert_charp_to_array(h, hash, 1000);
    *exist=false;
    strcat(sql, "SELECT id FROM position WHERE 1=1 ");
    for(int i=0;i<26;i++){
        sprintf(t, "and p%i=%i ",i, p->checker[i]);
        strcat(sql, t);
    }
    sprintf(t, "and player1_score=%i and player2_score=%i ",
            p->p1_score, p->p2_score);
    strcat(sql, t);
    sprintf(t, "and ((dice1=%i and dice2=%i) or (dice1=%i and dice2=%i)) ",
            p->dice[0],p->dice[1],p->dice[1],p->dice[0]);
    strcat(sql,t);
    sprintf(t,"and cube_position=%i and player_on_roll=%i and cube_action=%i;",
            p->cube, p->player_on_roll, p->cube_action);
    strcat(sql,t);
    printf("sql: %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }
    *nb=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        id[*nb]=sqlite3_column_int(stmt,0);
        *nb+=1;
        *exist=true;
    }
    return 1;
}

int db_insert_position_to_library(sqlite3* db,
        const int pos_id, const char *lib_name){
    printf("\ndb_insert_position_to_library\n");
    char sql[10000], t[10000]; int lib_id;
    sql[0]='\0'; t[0]='\0';
    db_get_library_id_from_name(db,lib_name,&lib_id);
    strcat(sql, "INSERT INTO catalog ");
    strcat(sql, "(position_id, library_id) ");
    sprintf(t, "VALUES (%d, %d);", pos_id, lib_id); 
    strcat(sql, t);
    execute_sql(db, sql);
    return 1;
}

int db_insert_library(sqlite3* db, const char *lib_name){
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0';
    printf("\ndb_insert_library\n");
    strcat(sql, "INSERT INTO library (name) ");
    sprintf(t, "VALUES (\"%s\");", lib_name);
    strcat(sql, t);
    execute_sql(db, sql);
    return 1;
}

bool db_is_position_in_library(sqlite3* db, int pos_id,
        const char *lib_name){
    printf("\ndb_is_position_in_library\n");
    char sql[10000]; int lib_id;
    db_get_library_id_from_name(db,lib_name,&lib_id);
    sprintf(sql, "SELECT count(*) FROM catalog WHERE position_id = %i and library_id = %i;",
            pos_id, lib_id);
    printf("sql %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    int n=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        n=sqlite3_column_int(stmt,0);
    }
    printf("n: %i\n", n);
    if(n>0) {return true;} else {return false;}
}

int db_select_libraries(sqlite3* db,
        int* lib_nb, int* lib_list_id,
        char lib_list[LIBRARIES_NUMBER_MAX][LIBRARY_NAME_MAX]){
    printf("\ndb_select_libraries\n");
    char sql[10000];
    sprintf(sql, "SELECT id,name FROM library");
    printf("sql %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
        return 0;
    }
    *lib_nb=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        lib_list_id[*lib_nb]=sqlite3_column_int(stmt,0);
        char *name=(char *)sqlite3_column_text(stmt,1);
        lib_list[*lib_nb][0]='\0';
        strcat(lib_list[*lib_nb], name);
        *lib_nb+=1;
    }
    if(rc!=SQLITE_DONE){
        printf("Failed to execute statement: %s\n",
                sqlite3_errmsg(db));
        return 0;
    }
    sqlite3_finalize(stmt);
    return 1;
}

int db_delete_library(sqlite3* db, const char* name){
    printf("\ndb_delete_library\n");
    if(!db_library_exists(db, name)) return 0;
    int id;
    printf("lib %s exists.\n",name);
    db_get_library_id_from_name(db, name, &id);
    printf("lib id %i\n",id);
    char sql[10000]; sql[0]='\0';
    sprintf(sql,"DELETE FROM catalog WHERE library_id = %i;",id);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    sprintf(sql,"DELETE FROM library WHERE id = %i;",id);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}

int db_rename_library(sqlite3* db, const char* old, const char* new){
    printf("\ndb_rename_library\n");
    printf("old new %s %s\n", old, new);
    char sql[10000]; sql[0]='\0';
    sprintf(sql, "UPDATE library SET name = \"%s\" WHERE name = \"%s\";",
            new, old);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}

int db_copy_library(sqlite3* db, const char* old, const char* new){
    printf("\ndb_copy_library\n");
    printf("old new %s %s\n", old, new);
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0'; int old_id, new_id;
    if(!db_library_exists(db,new)) db_delete_library(db,new);
    db_insert_library(db,new);
    db_get_library_id_from_name(db,old,&old_id);
    db_get_library_id_from_name(db,new,&new_id);
    //copier tous les registrer existants
    strcat(sql,"INSERT INTO catalog (position_id, library_id) SELECT  ");
    sprintf(t,"position_id,\"%d\" FROM catalog WHERE library_id = %d;",
            new_id, old_id);
    strcat(sql,t);
    printf("sql %s\n",sql);
    execute_sql(db,sql);
    return 1;
}

int db_get_libraries_related_to_position(sqlite3* db,
        const int pos_id, int* lname_nb,
        char lname_list[LIBRARIES_NUMBER_MAX][LIBRARY_NAME_MAX]){
    printf("\ndb_get_libraries_related_to_position\n");
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0';
    strcat(sql,"SELECT name FROM library l ");
    strcat(sql,"INNER JOIN catalog c ON l.id=c.library_id ");
    sprintf(t,"WHERE c.position_id=%d ;",pos_id);
    strcat(sql,t);
    printf("sql %s\n",sql);
    *lname_nb=0;
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        char *name=(char *)sqlite3_column_text(stmt,0);
        lname_list[*lname_nb][0]='\0';
        strcat(lname_list[*lname_nb],name);
        *lname_nb+=1;
    }
    sqlite3_finalize(stmt);
    return 1;
}

int db_select_position_from_library(sqlite3* db,
        const char* lname, int pid_list[POSITION_NUMBER_MAX], int* pid_nb){
    printf("\ndb_select_position_from_library\n");
    char sql[10000], t[10000]; sql[0]='\0'; t[0]='\0'; int lid;
    db_get_library_id_from_name(db,lname,&lid);
    strcat(sql,"SELECT id FROM position p ");
    strcat(sql,"INNER JOIN catalog c ON p.id=c.position_id ");
    sprintf(t,"WHERE c.library_id=%d ;",lid);
    strcat(sql,t);
    printf("sql %s\n",sql);
    *pid_nb=0;
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        pid_list[*pid_nb]=sqlite3_column_int(stmt,0);
        *pid_nb+=1;
    }
    sqlite3_finalize(stmt);
    printf("pid_nb %i\n",pid_nb);
    return 1;
}

int db_remove_position_from_library(sqlite3* db,
        const int pos_id, const char* l){
    printf("\ndb_remove_position_from_library\n");
    int lid; char sql[10000], t[100]; sql[0]='\0'; t[0]='\0';
    db_get_library_id_from_name(db,l,&lid);
    strcat(sql, "DELETE FROM catalog WHERE ");
    sprintf(t, "library_id = %i and position_id = %i ;",lid, pos_id);
    strcat(sql,t);
    printf("sql %s\n",sql); 
    execute_sql(db,sql);
    return 1;
}

int db_remove_position_from_libraries(sqlite3* db,
        const int pid){
    printf("\ndb_remove_position_from_libraries\n");
    int lname_nb; char lname_list[LIBRARIES_NUMBER_MAX][LIBRARY_NAME_MAX];
    db_get_libraries_related_to_position(db, pid, &lname_nb, lname_list);
    for(int i=0;i<lname_nb;i++){
        db_remove_position_from_library(db,pid,lname_list[i]);
    }
    return 1;
}

int db_delete_library_if_void(sqlite3* db, const char* l){
    printf("\ndb_delete_library_if_void\n");
    int pid_nb; int pid_list[POSITION_NUMBER_MAX];
    db_select_position_from_library(db, l, pid_list, &pid_nb);
    if(pid_nb==0) db_delete_library(db,l);
    return 1;
}

void parse_checker_analysis(char *l, CHECKER_ANALYSIS *a)
{
    char _t[LINE_LENGTHMAX]; _t[0]='\0';
    int n=strlen(l);
    printf("l: %s\n",l);
    printf("len l: %i\n",n);
    l+=7;
    strncpy(a->depth,l,11);
    l+=12;
    strncpy(_t,l,28);
    int i=0; char* token[10];
    char *c = strtok(_t, " ");
    while(c!=NULL){
        token[i]=c; i+=1;
        c=strtok(NULL, " ");
    }
    printf("parse checker analysis token nb: %i\n",i);
    for(int k=0;k<i;k++) printf("tok i %s %i\n",token[k],k);
    a->move1[0]='\0'; strcat(a->move1,token[0]);
    if(i>=2){
        a->move2[0]='\0';
        strcat(a->move2,token[1]);
    }
    if(i>=3){
        a->move3[0]='\0';
        strcat(a->move3,token[2]);
    }
    if(i>=4){
        a->move4[0]='\0';
        strcat(a->move4,token[3]);
    }

    l+=32;
    if(parse_from_file) l+=1; //extra char from text file
    if(n<=60){
        strncpy(_t,l,6);
        sscanf(_t,"%lf",&a->equity);
    } else {
        strncpy(_t,l,15);
        sscanf(_t,"%lf (%lf)",&a->equity,&a->error);
    }
    printf("equity: %f\n",a->equity);
    printf("error: %f\n",a->error);
}

void parse_line(const char *line, POSITION *p,
        int *index, CHECKER_ANALYSIS *a, CUBE_ANALYSIS *d,
        METADATA *m,
        bool *has_xgid, bool *has_playername, bool *has_score,
        bool *has_cube,
        bool *has_dice_action, bool *has_cube_action,
        bool *has_analysis1,
        bool *has_checker_p1wins, bool *has_checker_p2wins,
        bool *has_cube_p1wins, bool *has_cube_p2wins,
        bool *has_cubeless_eq, bool *has_cubeful_nd,
        bool *has_cubeful_dt, bool *has_cubeful_dp,
        bool *has_best_cube_action, bool *lookfor_rollout_wins_nd
        ){
    int p1_abs_score, p2_abs_score, match_point_nb,
    cube_value, roll;
    char _t[LINE_LENGTHMAX]; _t[0]='\0';
    char *l = malloc(strlen(line)+1);
    if(l) strcpy(l,line);

    printf("l: %s\n",l);

    char *_ptr=strstr(l,"XGID");
    if(_ptr!=NULL){
        l=_ptr;
        sscanf(l,"XGID=%s",m->xgid);
        *has_xgid=true;
    } else if(strncmp(l,"X:",2)==0 && strstr(l,"O:")!=0){
        sscanf(l,"X:%s   O:%s",m->p1_name,m->p2_name);
        *has_playername=true;
    } else if(strncmp(l,"Score is",8)==0){
        sscanf(l,"Score is X:%d O:%d %d pt.(s) match.",
                &p1_abs_score,&p2_abs_score,&match_point_nb);
        p->p1_score=match_point_nb-p1_abs_score;
        p->p2_score=match_point_nb-p2_abs_score;
        *has_score=true;
    } else if(strncmp(l,"Le score est",12)==0){
        sscanf(l,"Le score est X:%d O:%d match en %d pt(s)",
                &p1_abs_score,&p2_abs_score,&match_point_nb);
        p->p1_score=match_point_nb-p1_abs_score;
        p->p2_score=match_point_nb-p2_abs_score;
        *has_score=true;
    } else if(strncmp(l,"Cube: 1",7)==0
            && strstr(l,"own cube")==NULL){
        p->cube=0;
        *has_cube=true;
    } else if(strncmp(l,"Videau: 1",9)==0
            && strstr(l,"a le videau")==NULL){
        p->cube=0;
        *has_cube=true;
    } else if(strncmp(l,"Cube:",5)==0
            && strstr(l,"O own cube")!=NULL){
        sscanf(l,"Cube: %d, O own cube",&cube_value);
        p->cube=-1*((int)log2(cube_value));
        *has_cube=true;
    } else if(strncmp(l,"Videau:",7)==0
            && strstr(l,"O a le videau")!=NULL){
        sscanf(l,"Videau: %d, O a le videau",&cube_value);
        p->cube=-1*((int)log2(cube_value));
        *has_cube=true;
    } else if(strncmp(l,"Cube:",5)==0
            && strstr(l,"X own cube")!=NULL){
        sscanf(l,"Cube: %d, X own cube",&cube_value);
        p->cube=(int)log2(cube_value);
        *has_cube=true;
    } else if(strncmp(l,"Videau:",7)==0
            && strstr(l,"X a le videau")!=NULL){
        sscanf(l,"Videau: %d, X a le videau",&cube_value);
        p->cube=(int)log2(cube_value);
        *has_cube=true;
    } else if(strncmp(l,"X to play",9)==0){
        sscanf(l,"X to play %d",&roll);
        p->cube_action=0;
        p->dice[0]=roll/10;
        p->dice[1]=(int) fmod((double)roll,10.); // -lm?
        *has_dice_action=true;
    } else if(strstr(l,"jouer")!=0){
        l+=10;
        sscanf(l,"%d",&roll);
        p->cube_action=0;
        p->dice[0]=roll/10;
        p->dice[1]=(int) fmod((double)roll,10.); // -lm?
        *has_dice_action=true;
    } else if(strncmp(l,"X on roll, cube action",22)==0
            || strstr(l,"lance ou double")!=NULL){
        p->cube_action=1;
        *has_cube_action=true;
    } else if(strncmp(l,"Analyzed in",11)==0){
        l+=12;
        strncpy(d->depth,l,15);
    } else if(strstr(l,"Analys")!=0){
        l+=13;
        strncpy(d->depth,l,15);
    } else if(strncmp(l,"No double",9)==0
            || strncmp(l,"Pas de double",13)==0){
        *lookfor_rollout_wins_nd=true;
        printf("_rollout_wins_nd %i\n",*lookfor_rollout_wins_nd);
    } else if((strncmp(l,"  Player Winning Chances",24)==0 
            || strncmp(l,"  Chance de gain du joueur",26)==0)
            && *lookfor_rollout_wins_nd){
        printf("KOKOO\n");
        l+=28;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",&d->p1_w,&d->p1_g,&d->p1_b);
        *has_cube_p1wins=true;
    } else if(strncmp(l,"  Opponent Winning Chances",26)==0 && *lookfor_rollout_wins_nd){
        l+=28;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",&d->p2_w,&d->p2_g,&d->p2_b);
        *has_cube_p2wins=true;
        *lookfor_rollout_wins_nd=false;
        printf("_rollout_wins_nd %i\n",*lookfor_rollout_wins_nd);
    } else if(strncmp(l,"  Chance de gain de l'adversaire",32)==0
            && *lookfor_rollout_wins_nd){
        l+=34;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",&d->p2_w,&d->p2_g,&d->p2_b);
        *has_cube_p2wins=true;
        *lookfor_rollout_wins_nd=false;
    } else if(strncmp(l,"Player Winning Chances",22)==0
            || strncmp(l,"Chance de gain du joueur",24)==0){
        l+=26;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",&d->p1_w,&d->p1_g,&d->p1_b);
        *has_cube_p1wins=true;
    } else if(strncmp(l,"Opponent Winning Chances",24)==0){
        l+=26;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",&d->p2_w,&d->p2_g,&d->p2_b);
        *has_cube_p2wins=true;
    } else if(strncmp(l,"Chance de gain de l'adversaire",30)==0){
        l+=32;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",&d->p2_w,&d->p2_g,&d->p2_b);
        *has_cube_p2wins=true;
    } else if(strncmp(l,"Cubeless Equities",17)==0){
        l+=19;
        strncpy(_t,l,32);
        sscanf(_t,"No Double=%lf, Double=%lf",
                &d->cubeless_equity_nd,&d->cubeless_equity_d);
        *has_cubeless_eq=true;
    } else if(strstr(l,"sans videau")!=0){
        l+=21;
        strncpy(_t,l,32);
        sscanf(_t,"Pas de double=%lf, Double=%lf",
                &d->cubeless_equity_nd,&d->cubeless_equity_d);
        *has_cubeless_eq=true;
    } else if(strncmp(l,"       No double",16)==0
            || strncmp(l,"       Pas de double",20)==0){
        l+=22;
        if(strstr(l,"(")==NULL){
            strncpy(_t,l,6);
            sscanf(_t,"%lf",&d->cubeful_equity_nd);
            d->error_nd=0;
        } else {
            strncpy(_t,l,15);
            sscanf(_t,"%lf (%lf)",&d->cubeful_equity_nd,&d->error_nd);
        }
        *has_cubeful_nd=true;
    } else if(strncmp(l,"       No redouble",18)==0
            || strncmp(l,"       Pas de redouble",22)==0){
        l+=24;
        if(strstr(l,"(")==NULL){
            strncpy(_t,l,6);
            sscanf(_t,"%lf",&d->cubeful_equity_nd);
            d->error_nd=0;
        } else {
            strncpy(_t,l,15);
            sscanf(_t,"%lf (%lf)",&d->cubeful_equity_nd,&d->error_nd);
        }
        *has_cubeful_nd=true;
    } else if(strncmp(l,"       Double/Take",18)==0
            || strncmp(l,"       Double/Prend",19)==0){
        l+=22;
        if(strstr(l,"(")==NULL){
            strncpy(_t,l,6);
            sscanf(_t,"%lf",&d->cubeful_equity_dt);
            d->error_dt=0;
        } else {
            strncpy(_t,l,15);
            sscanf(_t,"%lf (%lf)",&d->cubeful_equity_dt,&d->error_dt);
        }
        *has_cubeful_dt=true;
    } else if(strncmp(l,"       Redouble/Take",20)==0
            || strncmp(l,"       Redouble/Beaver",22)==0
            || strncmp(l,"       Redouble/Prend",21)==0){
        if(strstr(l,"Beaver")!=0){
            d->beaver=1;
        }else{ d->beaver=0;}
        l+=24;
        if(strstr(l,"(")==NULL){
            strncpy(_t,l,6);
            sscanf(_t,"%lf",&d->cubeful_equity_dt);
            d->error_dt=0;
        } else {
            strncpy(_t,l,15);
            sscanf(_t,"%lf (%lf)",&d->cubeful_equity_dt,&d->error_dt);
        }
        *has_cubeful_dt=true;
    } else if(strncmp(l,"       Double/Pass",18)==0
            || strncmp(l,"       Double/Passe",19)==0){
        l+=22;
        if(strstr(l,"(")==NULL){
            strncpy(_t,l,6);
            sscanf(_t,"%lf",&d->cubeful_equity_dp);
            d->error_dp=0;
        } else {
            strncpy(_t,l,15);
            sscanf(_t,"%lf (%lf)",&d->cubeful_equity_dp,&d->error_dp);
        }
        *has_cubeful_dp=true;
    } else if(strncmp(l,"       Redouble/Pass",20)==0
            || strncmp(l,"       Redouble/Passe",21)==0){
        l+=24;
        if(strstr(l,"(")==NULL){
            strncpy(_t,l,6);
            sscanf(_t,"%lf",&d->cubeful_equity_dp);
            d->error_dp=0;
        } else {
            strncpy(_t,l,15);
            sscanf(_t,"%lf (%lf)",&d->cubeful_equity_dp,&d->error_dp);
        }
        *has_cubeful_dp=true;
    } else if(strncmp(l,"Best Cube action",16)==0){
        if(strstr(l,"No double / Take")!=NULL) d->best_cube_action=0;
        if(strstr(l,"Double / Take")!=NULL) d->best_cube_action=1;
        if(strstr(l,"Double / Pass")!=NULL) d->best_cube_action=2;
        if(strstr(l,"Redouble / Take")!=NULL) d->best_cube_action=1;
        if(strstr(l,"Redouble / Pass")!=NULL) d->best_cube_action=2;
        if(strstr(l,"Too good to double / Pass")!=NULL) d->best_cube_action=3;
        if(strstr(l,"Too good to double / Take")!=NULL) d->best_cube_action=4;
        *has_best_cube_action=true;
    } else if(strncmp(l,"Meilleur action du videau",25)==0){
        if(strstr(l,"Pas de double / Prend")!=NULL) d->best_cube_action=0;
        if(strstr(l,"Double / Prend")!=NULL) d->best_cube_action=1;
        if(strstr(l,"Double / Passe")!=NULL) d->best_cube_action=2;
        if(strstr(l,"Redouble / Prend")!=NULL) d->best_cube_action=1;
        if(strstr(l,"Redouble / Passe")!=NULL) d->best_cube_action=2;
        if(strstr(l,"Trop bon pour doubler / Passe")!=NULL) d->best_cube_action=3;
        if(strstr(l,"Trop bon pour doubler / Prend")!=NULL) d->best_cube_action=4;
        *has_best_cube_action=true;
    } else if(strncmp(l,"Percentage of wrong take",24)==0){
        l+=67;
        sscanf(l,"%lf%%",&d->percentage_wrong_pass_make_good_double);
    } else if(strncmp(l,"Pourcentage de prises incorrectes",33)==0){
        l+=78;
        sscanf(l,"%lf%%",&d->percentage_wrong_pass_make_good_double);
    } else if(strncmp(l,"eXtreme",7)==0){
        strncpy(m->misc,l,47);
        m->misc[47]='\0';
    } else if(strncmp(l,"    1.",6)==0){
        *index=0; parse_checker_analysis(l,&a[*index]);
        *has_analysis1=true;
    } else if(strncmp(l,"    2.",6)==0){                 
        *index=1; parse_checker_analysis(l,&a[*index]);
    } else if(strncmp(l,"    3.",6)==0){                 
        *index=2; parse_checker_analysis(l,&a[*index]);
    } else if(strncmp(l,"    4.",6)==0){                 
        *index=3; parse_checker_analysis(l,&a[*index]);
    } else if(strncmp(l,"    5.",6)==0){
        *index=4; parse_checker_analysis(l,&a[*index]);
    } else if(strncmp(l,"      Player",12)==0){
        l+=16;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",
                &a[*index].p1_w,&a[*index].p1_g,&a[*index].p1_b);
        *has_checker_p1wins=true;
    } else if(strncmp(l,"      Joueur",12)==0){
        l+=18;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",
                &a[*index].p1_w,&a[*index].p1_g,&a[*index].p1_b);
        *has_checker_p1wins=true;
    } else if(strncmp(l,"      Opponent",14)==0){
        l+=16;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",
                &a[*index].p2_w,&a[*index].p2_g,&a[*index].p2_b);
        *has_checker_p2wins=true;
    } else if(strncmp(l,"      Adversaire",16)==0){
        l+=18;
        strncpy(_t,l,30);
        sscanf(_t,"%lf%% (G:%lf%% B:%lf%%)",
                &a[*index].p2_w,&a[*index].p2_g,&a[*index].p2_b);
        *has_checker_p2wins=true;
    }

    free(l);
}


int db_import_position_from_lines(sqlite3* db,
        char lines[LINE_NBMAX][LINE_LENGTHMAX], int line_nb, int* pid){
    printf("\ndb_import_position_from_lines\n");
    POSITION p;
    CHECKER_ANALYSIS a[5]; int ca_index=0;
    CUBE_ANALYSIS d;
    METADATA m;
    int p1_abs_score,p2_abs_score,match_point_nb;
    int cube_value, cube_owner, roll;
    bool exist;
    int nb, _id[1000];
    bool has_xgid, has_playername, has_score,
         has_cube,
         has_dice_action, has_cube_action,
         has_analysis1, 
         has_checker_p1wins, has_checker_p2wins,
         has_cube_p1wins, has_cube_p2wins,
         has_cubeless_eq, has_cubeful_nd,
         has_cubeful_dt, has_cubeful_dp,
         has_best_cube_action;
    bool has_valid_position, has_valid_checker_analysis,
         has_valid_cube_analysis, is_valid_position_file;
    bool lookfor_rollout_wins_nd;

    p=POS_VOID;
    for(int i=0;i<5;i++) a[i]=CA_VOID;
    d=D_VOID;
    m=M_VOID;

    has_xgid=false;
    has_playername=false;
    has_score=false;
    has_cube=false;
    has_dice_action=false;
    has_cube_action=false;

    has_analysis1=false;
    has_checker_p1wins=false;
    has_checker_p2wins=false;

    has_cube_p1wins=false;
    has_cube_p2wins=false;
    has_cubeless_eq=false;
    has_cubeful_nd=false;
    has_cubeful_dt=false;
    has_cubeful_dp=false;
    has_best_cube_action=false;
    lookfor_rollout_wins_nd=false;

    for(int i=0;i<line_nb;i++){
        parse_line(lines[i],&p,&ca_index,a,&d,&m,
                &has_xgid, &has_playername, &has_score,
                &has_cube,
                &has_dice_action, &has_cube_action,
                &has_analysis1,
                &has_checker_p1wins, &has_checker_p2wins,
                &has_cube_p1wins, &has_cube_p2wins,
                &has_cubeless_eq, &has_cubeful_nd,
                &has_cubeful_dt, &has_cubeful_dp,
                &has_best_cube_action,
                &lookfor_rollout_wins_nd
                );
    }

    has_valid_position = has_xgid && has_playername
        && has_score && has_cube 
        && (has_dice_action || has_cube_action);

    printf("has_xgid: %i\n", has_xgid);
    printf("has_playername: %i\n", has_playername);
    printf("has_score: %i\n", has_score);
    printf("has_cube: %i\n", has_cube);
    printf("has_dice_action: %i\n", has_dice_action);
    printf("has_cube_action: %i\n", has_cube_action);
    printf("has_best_cube_action: %i\n", has_best_cube_action);
    printf("has_cube_p1wins: %i\n", has_cube_p1wins);
    printf("has_cube_p2wins: %i\n", has_cube_p2wins);
    printf("has_cubeless_eq: %i\n", has_cubeless_eq);
    printf("has_cubeful_nd: %i\n", has_cubeful_nd);
    printf("has_cubeful_dt: %i\n", has_cubeful_dt);
    printf("has_cubeful_dp: %i\n", has_cubeful_dp);

    has_valid_checker_analysis = has_analysis1 
        && has_checker_p1wins && has_checker_p2wins;

    has_valid_cube_analysis = has_cube_p1wins
        && has_cube_p2wins && has_cubeless_eq
        && has_cubeful_nd && has_cubeful_dt && has_cubeful_dp
        && has_best_cube_action;

    is_valid_position_file = has_valid_position;

    printf("is_valid_position_file: %i\n", is_valid_position_file);
    printf("has_valid_position: %i\n", has_valid_position);
    printf("has_valid_checker_analysis: %i\n", has_valid_checker_analysis);
    printf("has_valid_cube_analysis: %i\n", has_valid_cube_analysis);

    if(!is_valid_position_file) return 0;

    // checker if all information are gathered and position is valid
    // has_dice_action or has_cube_action. disfonction de cas

    if(has_valid_checker_analysis){
        for(int i=0;i<5;i++)
            checker_analysis_print(&a[i]);
    }
    if(has_valid_cube_analysis) cube_analysis_print(&d);
    metadata_print(&m);

    convert_xgid_to_position(m.xgid,&p);
    position_print(&p);

    db_find_identical_position(db, &p, &exist, &nb, _id);
    if(!exist){
        printf("position does not exist\n");
        db_insert_position(db,&p);
        *pid=db_last_insert_id(db,"position");
    } else { *pid=_id[0]; 
        printf("position does exist\n");
    }
    db_delete_if_exist_metadata(db,*pid);
    db_insert_metadata(db,pid,&m);
    if(has_valid_checker_analysis){
        db_delete_if_exist_checker_analysis(db,*pid);
        for(int i=0;i<5;i++)
            db_insert_checker_analysis(db,pid,&a[i]);
    } else if (has_valid_cube_analysis){
        db_delete_if_exist_cube_analysis(db,*pid);
        db_insert_cube_analysis(db,pid,&d);
    }

    return 1;
}

int db_import_position_from_file(sqlite3* db, FILE* f, int* pid){
    printf("\ndb_import_position_from_file\n");
    char lines[LINE_NBMAX][LINE_LENGTHMAX]; int nb;
    nb=0;
    lines[nb][0]='\0';
    while(fgets(lines[nb],sizeof(lines[nb]),f)){
        lines[nb][strcspn(lines[nb],"\r\n")]=0; //delete \n end of line
        nb+=1;
        lines[nb][0]='\0';
    }
    return db_import_position_from_lines(db,lines,nb,pid);
}

int isBlankLine(const char *str) {
    if (str == NULL || str[0] == '\0') {
        return 1;
    }
    for (int i = 0; i < strlen(str); i++) {
        if (!isspace((unsigned char)str[i])) {
            return 0;
        }
    }
    return 1;
}

void parse_match_move(char *s, MOVE *m){
    printf("\nparse_match_move\n");
    if(strstr(s,":")!=0){ //checker move
                           //parse dice
        int roll=0;
        char _h[5]; _h[0]='\0';
        strncpy(_h,s,2);
        sscanf(_h,"%d",roll);
        m->dice[0]=roll/10;
        m->dice[1]=(int) fmod((double)roll,10.);
        s+=3; //parse checker move
        int _i=0; char* token[10];
        char *c = strtok(s, " ");
        while(c!=NULL){
            token[_i]=c; _i+=1;
            c=strtok(NULL, " ");
        }
        printf("parse checker play token nb: %i\n",_i);
        for(int k=0;k<_i;k++) printf("tok i %s %i\n",token[k],k);
        strcat(m->move1,token[0]);
        if(_i>=2){
            strcat(m->move2,token[1]);
        }
        if(_i>=3){
            strcat(m->move3,token[2]);
        }
        if(_i>=4){
            strcat(m->move4,token[3]);
        }
    } else if(strstr(s,"Doubles =>")!=0
            || strstr(s,"Takes")!=0
            || strstr(s, "Drops")!=0){ //cube move
        strcat(m->move1, s);
    }
}

int db_import_match_from_file(sqlite3* db, FILE* f){
    printf("\ndb_import_from_file\n");
    char lines[LINE_NBMAX][LINE_LENGTHMAX]; int nb;
    char _s[LINE_LENGTHMAX]; _s[0]='\0';
    nb=0; lines[nb][0]='\0';
    GAME games[50]; MOVE moves[50][1000];
    int moves_per_game[50]; int move_nb=0;
    int game_nb=0; int matchLength=0;
    char site[100]; site[0]='\0';
    char match_id[100]; match_id[0]='\0';
    char event[100]; event[0]='\0';
    char player1_name[100]; player1_name[0]='\0';
    char player2_name[100]; player2_name[0]='\0';
    char eventDate[100]; eventDate[0]='\0';
    char eventTime[100]; eventTime[0]='\0';
    char clockType[100]; clockType[0]='\0';
    bool is_valid_match=false;
    bool is_valid_game=false;
    bool is_parsing_game=false;

    for(int i=0;i<50;i++){
        games[i]=G_VOID;
        for(int j=0;j<1000;j++){
            moves[i][j]=MV_VOID;
        }
    }


    while(fgets(lines[nb],sizeof(lines[nb]),f)){
        lines[nb][strcspn(lines[nb],"\r\n")]=0; //delete \n end of line
        nb+=1;
        lines[nb][0]='\0';
    }

    char *l = malloc(LINE_LENGTHMAX+1);
    char *_ptr;
    for(int i=0;i<nb;i++){
        if(l) strcpy(l,lines[i]);

        _ptr=strstr(l,"[Site");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[Site \"%s\"]",site);
        }

        _ptr=strstr(l,"[Match");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[Match ID \"%s\"]",match_id);
        }

        _ptr=strstr(l,"[Event");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[Event \"%s\"]",event);
        }

        _ptr=strstr(l,"[Player 1");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[Player 1 \"%s\"]",player1_name);
        }

        _ptr=strstr(l,"[Player 2");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[Player 2 \"%s\"]",player2_name);
        }

        _ptr=strstr(l,"[EventDate");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[EventDate \"%s\"]",eventDate);
        }

        _ptr=strstr(l,"[EventTime");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[EventTime \"%s\"]",eventTime);
        }

        _ptr=strstr(l,"[ClockType");
        if(_ptr!=NULL){
            l=_ptr;
            sscanf(l,"[ClockType \"%s\"]",clockType);
        }

        _ptr=strstr(l,"point match");
        if(_ptr!=NULL){
            sscanf(l,"%d point match",matchLength);
        }

        _ptr=strstr(l,"Game");
        if(_ptr!=NULL){
            game_nb+=1;
            move_nb=0;
            is_parsing_game=true;
        }

        if(strncmp(l,"  ",2)==0 && isdigit(l[2])){
            _ptr=strstr(l,") ");
            l=_ptr+2; //move player1
            strncpy(_s,l,30);
            if(!isBlankLine(_s)){
                parse_match_move(_s, &moves[game_nb-1][move_nb]);
                moves[game_nb-1][move_nb].player_on_roll=PLAYER1;
                move_nb+=1;
            }
            l+=35; //move player2
           strncpy(_s,l,30);
            if(!isBlankLine(_s)){
                parse_match_move(_s, &moves[game_nb-1][move_nb]);
                    moves[game_nb-1][move_nb].player_on_roll=PLAYER2;
                move_nb+=1;
            }
        }

        _ptr=strstr(l,"Wins");
        if(_ptr!=NULL){
            moves_per_game[game_nb-1]=move_nb;
            is_parsing_game=false;
        }

    }
    free(l);
    is_valid_match=false;

    return 1;
}

int db_select_checker_analysis(sqlite3* db, int pid,
        CHECKER_ANALYSIS* ca, int *ca_nb){
    printf("\ndb_select_checker_analysis\n");
    char sql[10000]; sql[0]='\0';
    int i=0;
    sprintf(sql,"SELECT * FROM checker_analysis WHERE position_id=%d;",pid);
    printf("sql %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }
    *ca_nb=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        ca[*ca_nb].depth[0]='\0';
        strcat(ca[*ca_nb].depth, sqlite3_column_text(stmt,1));
        ca[*ca_nb].move1[0]='\0';
        strcat(ca[*ca_nb].move1, sqlite3_column_text(stmt,2));
        ca[*ca_nb].move2[0]='\0';
        strcat(ca[*ca_nb].move2, sqlite3_column_text(stmt,3));
        ca[*ca_nb].move3[0]='\0';
        strcat(ca[*ca_nb].move3, sqlite3_column_text(stmt,4));
        ca[*ca_nb].move4[0]='\0';
        strcat(ca[*ca_nb].move4, sqlite3_column_text(stmt,5));
        ca[*ca_nb].equity=sqlite3_column_double(stmt,6);
        ca[*ca_nb].error=sqlite3_column_double(stmt,7);
        ca[*ca_nb].p1_w=sqlite3_column_double(stmt,8);
        ca[*ca_nb].p1_g=sqlite3_column_double(stmt,9);
        ca[*ca_nb].p1_b=sqlite3_column_double(stmt,10);
        ca[*ca_nb].p2_w=sqlite3_column_double(stmt,11);
        ca[*ca_nb].p2_g=sqlite3_column_double(stmt,12);
        ca[*ca_nb].p2_b=sqlite3_column_double(stmt,13);
        *ca_nb+=1;
    }
    sqlite3_finalize(stmt);
    printf("ca_nb %i\n",*ca_nb);
    return 1;
}

int db_select_cube_analysis(sqlite3* db, int pid,
        CUBE_ANALYSIS* da, int *da_nb){
    printf("\ndb_select_cube_analysis\n");
    char sql[10000]; sql[0]='\0';
    int i=0;
    sprintf(sql,"SELECT * FROM cube_analysis WHERE position_id=%d;",pid);
    printf("sql %s\n", sql);
    int rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    if(rc!=SQLITE_OK){
        printf("Failed to prepare statement: %s\n",
                sqlite3_errmsg(db));
    }
    *da_nb=0;
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        da->depth[0]='\0';
        strcat(da->depth, sqlite3_column_text(stmt,1));
        da->p1_w=sqlite3_column_double(stmt,2);
        da->p1_g=sqlite3_column_double(stmt,3);
        da->p1_b=sqlite3_column_double(stmt,4);
        da->p2_w=sqlite3_column_double(stmt,5);
        da->p2_g=sqlite3_column_double(stmt,6);
        da->p2_b=sqlite3_column_double(stmt,7);
        da->cubeless_equity_nd=sqlite3_column_double(stmt,8);
        da->cubeless_equity_d=sqlite3_column_double(stmt,9);
        da->cubeful_equity_nd=sqlite3_column_double(stmt,10);
        da->cubeful_equity_dt=sqlite3_column_double(stmt,11);
        da->cubeful_equity_dp=sqlite3_column_double(stmt,12);
        da->error_nd=sqlite3_column_double(stmt,13);
        da->error_dt=sqlite3_column_double(stmt,14);
        da->error_dp=sqlite3_column_double(stmt,15);
        da->best_cube_action=sqlite3_column_int(stmt,16);
        da->percentage_wrong_pass_make_good_double=
            sqlite3_column_double(stmt,17);
        da->beaver=sqlite3_column_int(stmt,18);
        *da_nb+=1;
    }
    sqlite3_finalize(stmt);
    printf("da_nb %i\n",*da_nb);
    return 1;
}

int db_analysis_exist(sqlite3* db, int pid){
    printf("ńdb_analysis_exist\n");
    int exists=0;
    char sql[10000]; sql[0]='\0';
    int i=0; int rc;
    sprintf(sql,"SELECT * FROM cube_analysis WHERE position_id=%d;",pid);
    printf("sql %s\n", sql);
    rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        return 1;
    }
    sprintf(sql,"SELECT * FROM checker_analysis WHERE position_id=%d;",pid);
    printf("sql %s\n", sql);
    rc=sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);
    while((rc=sqlite3_step(stmt))==SQLITE_ROW){
        return 1;
    }
    sqlite3_finalize(stmt);
    return 0;
}

int db_insert_game(sqlite3 *db, const GAME *g,
        const int id_prev, const int id_next){
    printf("\ndb_insert_game\n");
    char _s[100];
    char sql[1000];
    sql[0]='\0';
    strcat(sql, "INSERT INTO game ");
    strcat(sql, "(site, match_id, event, player1_name, player2_name, ");
    strcat(sql, "eventDate, eventTime, clockType, matchLength, ");
    strcat(sql, "player1_score, player2_score, game_prev, game_next) ");
    strcat(sql, "VALUES ");
    strcat(sql, "(");
    sprintf(_s, "\"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", ",
            g->site, g->match_id, g->event, g->player1_name,
            g->player2_name, g->eventDate, g->eventTime);
    strcat(sql, _s);
    sprintf(_s, "\"%s\", %d, %d, %d, %d, %d); ",
            g->clockType, g->matchLength, g->player1_score,
            g->player2_score, id_prev, id_next);
    strcat(sql, _s);
    printf("sql insert: %s\n", sql);
    printf("Try to add new game.\n");
    execute_sql(db, sql); 
    return 1;
}

int db_insert_move(sqlite3 *db, const MOVE *g, const int game_id,
        const int id_prev, const int id_next, const int position_id){
    printf("\ndb_insert_move\n");
    char _s[100];
    char sql[1000];
    sql[0]='\0';
    strcat(sql, "INSERT INTO move ");
    strcat(sql, "(game_id, dice1, dice2, move1, move2, move3, move4, ");
    strcat(sql, "player_on_roll, move_prev, move_next, position_id) ");
    strcat(sql, "VALUES ");
    strcat(sql, "(");
    sprintf(_s, "%d, %d, %d, \"%s\", \"%s\", \"%s\", \"%s\", %d, %d, %d, %d);",
            game_id, g->dice[0], g->dice[1],
            g->move1, g->move2, g->move3, g->move4,
            g->player_on_roll, id_prev, id_next, position_id);
    strcat(sql, _s);
    printf("sql insert: %s\n", sql);
    printf("Try to add new move.\n");
    execute_sql(db, sql); 
    return 1;
}

/* END Database */



/************************ Interface ***********************/

/* BEGIN Interface */

/* #define DEFAULT_SIZE "960x540" */
/* #define DEFAULT_SIZE "864x486" */
/* #define DEFAULT_SIZE "800x486" */
#define DEFAULT_SIZE "800x440"
#define DEFAULT_SPLIT_VALUE "745"
#define DEFAULT_SPLIT_MINMAX "800:2000"
#define SB_DEFAULT_FONTSIZE "10" //sb=statusbar
#define ANALYSIS_MULTILINE_MAX 10000

enum mode { NORMAL, EDIT, CMD, MATCH };
typedef enum mode mode_t;
mode_t mode_active = NORMAL;

char lib_list[LIBRARIES_NUMBER_MAX][LIBRARY_NAME_MAX];
int lib_list_id[LIBRARIES_NUMBER_MAX];
int lib_index; //active library
int lib_nb;

bool make_point=true;
bool is_score_to_fill=false;
bool is_point_to_fill=false;
bool is_cube_to_fill=false;
int point_m, point_m2;
int key_m=-1;
int sign_m=1;
char digit_m[4];

char *cmdtext;
char* cmdtoken[100];
int token_nb;

char _c[100];

const char* msg_err_not_pos_nor_match_file =
"ERR: Text file is not a position nor a match.";
const char* msg_err_failed_to_import_pos =
"ERR: Failed to import position.";
const char* msg_err_failed_to_import_match =
"ERR: Failed to import match.";
const char* msg_err_failed_to_create_db =
"ERR: Failed to create database.";
const char* msg_err_invalid_library_name =
"ERR: Invalid library name. It must not start with \"-\" and contain only: alphanumeric symbols, \"-\", \"_\".";
const char* msg_err_no_db_opened =
"ERR: No database opened.";
const char* msg_err_failed_to_open_db =
"ERR: Failed to open database.";
const char* msg_err_cannot_update_position_with_analysis =
"ERR: Position with analysis cannot be updated.";
const char* msg_info_position_written = 
"Position written to database.";
const char* msg_info_position_updated = 
"Position updated.";
const char* msg_info_position_deleted = 
"Position deleted.";
const char* msg_info_position_imported = 
"Position imported.";
const char* msg_info_position_already_exists = 
"Position already exists in database.";
const char* msg_info_position_added_to_library =
"Position added to library.";
const char* msg_info_position_removed_from_library =
"Position removed from library.";
const char* msg_info_no_position =
"No positions.";
const char* msg_info_library_does_not_exist =
"Library does not exist.";
const char* msg_info_no_db_loaded =
"No database loaded.";
const char* msg_info_db_created =
"Database created.";
const char* msg_info_db_loaded =
"Database loaded.";
const char* msg_info_match_imported =
"Match imported.";

Ihandle *dlg, *menu, *toolbar, *position, *split, *searches, *statusbar;
Ihandle *cmdline, *analysis, *checker_analysis, *cube_analysis;
Ihandle *canvas, *search, *matchlib;
Ihandle *search1, *search2, *search3;
Ihandle *sb_mode, *sb_lib, *sb_msg; // sb=statusbar
Ihandle *hbox, *vbox, *lbl, *hspl, *vspl, *spl, *tabs, *txt;
bool is_searches_visible = false;

char* mode_to_str(const int mode) {
    char s[20]; s[0]='\0';
    switch(mode) {
        case NORMAL:
            return "NORMAL";
        case EDIT:
            return "EDIT";
        case CMD:
            return "COMMAND";
        case MATCH:
            return "MATCH";
        default:
            return "UNKNOWN";
    }
}

static Ihandle* create_menus(void)
{

    Ihandle *menu, *submenu_file, *submenu_edit,
            *submenu_position, *submenu_match,
            *submenu_tool, *submenu_help;

    Ihandle *menu_file;
    Ihandle *item_new, *item_open;
    Ihandle *item_import;
    Ihandle *item_save, *item_saveas;
    Ihandle *item_export;
    Ihandle *item_properties;
    Ihandle *item_exit;

    Ihandle *menu_edit;
    Ihandle *item_undo, *item_redo, *item_copy, *item_cut, *item_paste;
    Ihandle *item_board_direction_right, *item_board_direction_left;
    Ihandle *item_player_on_roll_down, *item_player_on_roll_up;
    Ihandle *item_editmode;

    Ihandle *menu_position;
    Ihandle *item_first_position, *item_last_position,
            *item_next_position, *item_prev_position,
            *item_new_position, *item_update_position,
            *item_import_position;
    Ihandle *item_new_library, *item_delete_library,
            *item_add_library, *item_goto_library;

    Ihandle *menu_match;
    Ihandle *item_import_match, *item_import_match_bybatch, 
            *item_match_library;

    Ihandle *item_search_engine;
    Ihandle *menu_tool;
    Ihandle *item_analysis, *item_find_position_without_analysis;
    Ihandle *item_preferences;

    Ihandle *menu_help;
    Ihandle *item_manual, *item_userguide, *item_tips, *item_cmdmode;
    Ihandle *item_keyboard;
    Ihandle *item_getinvolved, *item_donate;
    Ihandle *item_about;


    item_new = IupItem("&New Database\tCtrl+N", NULL);
    item_open = IupItem("&Open Database\tCtrl+O", NULL);
    item_save = IupItem("&Save Database", NULL);
    item_saveas = IupItem("Save &As...", NULL);
    item_import = IupItem("&Import...\tCtrl+I", NULL);
    item_export = IupItem("&Export...", NULL);
    item_properties = IupItem("Database &Metadata...", NULL);
    item_exit = IupItem("E&xit\tCtrl+Q", NULL);
    menu_file = IupMenu(item_new, item_open,
            IupSeparator(), item_import,
            IupSeparator(), item_exit, NULL);
    submenu_file = IupSubmenu("&File", menu_file);

    item_undo = IupItem("&Undo\tCtrl-Z", NULL);
    item_redo = IupItem("&Redo\tCtrl-Y", NULL);
    item_copy = IupItem("Co&py\tCtrl-C", NULL);
    item_cut = IupItem("Cu&t\tCtrl-X", NULL);
    item_paste = IupItem("Pa&ste\tCtrl-V", NULL);
    item_editmode = IupItem("&Edit Mode\tTab", NULL);
    item_board_direction_right = IupItem("Bear-off on the &right\tCtrl-Right", NULL);
    item_board_direction_left = IupItem("Bear-off on the &left\tCtrl-Left", NULL);
    item_player_on_roll_down = IupItem("Player on roll &down\tCtrl-Down", NULL);
    item_player_on_roll_up = IupItem("Player on roll &up\tCtrl-Up", NULL);
    menu_edit = IupMenu( item_copy, item_paste,
            IupSeparator(), item_board_direction_right,
            item_board_direction_left,
            item_player_on_roll_down, item_player_on_roll_up,
            IupSeparator(), item_editmode, NULL);
    submenu_edit = IupSubmenu("&Edit", menu_edit);

    item_first_position = IupItem("&First Position\tPg Up", NULL);
    item_last_position = IupItem("La&st Position\tPg Down", NULL);
    item_next_position = IupItem("Ne&xt Position\tj,Right", NULL);
    item_prev_position = IupItem("Pre&vious Position\tk,Left", NULL);
    item_new_position = IupItem("Ne&w Position", NULL);
    item_update_position = IupItem("&Update Position", NULL);
    item_import_position = IupItem("&Import Position", NULL);
    /* item_new_library = IupItem("New &Library", NULL); */
    item_add_library = IupItem("&Add Position to Library", NULL);
    item_goto_library = IupItem("&Go to Library", NULL);
    item_delete_library = IupItem("&Delete Library", NULL);
    menu_position = IupMenu(item_first_position,
            item_next_position, item_prev_position, item_last_position,
            item_new_position, item_update_position, IupSeparator(), item_import_position, 
            IupSeparator(),
            item_add_library, item_goto_library, item_delete_library,
            NULL);
    submenu_position = IupSubmenu("&Positions", menu_position);

    item_import_match = IupItem("&Import Match", NULL);
    item_import_match_bybatch = IupItem("Import Matches by &Batch",
            NULL);
    item_match_library = IupItem("Match &Library", NULL);
    menu_match = IupMenu(item_import_match, item_import_match_bybatch,
            item_match_library, NULL);
    submenu_match = IupSubmenu("&Matches", menu_match);


    item_analysis = IupItem("&Display Analysis\tCtrl-L", NULL);
    item_find_position_without_analysis = IupItem("&Find Positions without Analysis", NULL);
    item_search_engine = IupItem("Search &Engine\tCtrl-F", NULL);
    item_preferences = IupItem("&Preferences", NULL);
    menu_tool = IupMenu(item_analysis, item_find_position_without_analysis, IupSeparator(),
            item_search_engine, NULL);
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
            submenu_match, submenu_tool, submenu_help,
            NULL);

    IupSetHandle("menu", menu);

    IupSetCallback(item_new, "ACTION", (Icallback) item_new_action_cb);
    IupSetCallback(item_open, "ACTION", (Icallback) item_open_action_cb);
    IupSetCallback(item_import, "ACTION", (Icallback) item_import_action_cb);
    IupSetCallback(item_export, "ACTION", (Icallback) item_export_action_cb);
    IupSetCallback(item_save, "ACTION", (Icallback) item_save_action_cb);
    IupSetCallback(item_saveas, "ACTION", (Icallback) item_saveas_action_cb);
    IupSetCallback(item_properties, "ACTION", (Icallback) item_properties_action_cb);
    IupSetCallback(item_exit, "ACTION", (Icallback) item_exit_action_cb);
    IupSetCallback(item_copy, "ACTION", (Icallback) item_copy_action_cb);
    IupSetCallback(item_paste, "ACTION", (Icallback) item_paste_action_cb);
    IupSetCallback(item_editmode, "ACTION", (Icallback) item_editmode_action_cb);
    IupSetCallback(item_board_direction_left, "ACTION", (Icallback) board_direction_left_cb);
    IupSetCallback(item_board_direction_right, "ACTION", (Icallback) board_direction_right_cb);
    IupSetCallback(item_player_on_roll_up, "ACTION", (Icallback) display_player_on_roll_up);
    IupSetCallback(item_player_on_roll_down, "ACTION", (Icallback) display_player_on_roll_down);
    IupSetCallback(item_first_position, "ACTION", (Icallback) goto_first_position_cb);
    IupSetCallback(item_last_position, "ACTION", (Icallback) goto_last_position_cb);
    IupSetCallback(item_next_position, "ACTION", (Icallback) item_nextposition_action_cb);
    IupSetCallback(item_prev_position, "ACTION", (Icallback) item_prevposition_action_cb);
    IupSetCallback(item_new_position, "ACTION", (Icallback) item_newposition_action_cb);
    IupSetCallback(item_update_position, "ACTION", (Icallback) item_updateposition_action_cb);
    IupSetCallback(item_import_position, "ACTION", (Icallback) item_import_action_cb);
    IupSetCallback(item_new_library, "ACTION", (Icallback) item_newlibrary_action_cb);
    IupSetCallback(item_delete_library, "ACTION", (Icallback) item_deletelibrary_action_cb);
    IupSetCallback(item_add_library, "ACTION", (Icallback) item_addtolibrary_action_cb);
    IupSetCallback(item_goto_library, "ACTION", (Icallback) item_gotolibrary_action_cb);
    IupSetCallback(item_import_match, "ACTION", (Icallback) item_importmatch_action_cb);
    IupSetCallback(item_import_match_bybatch, "ACTION", (Icallback) item_importmatchbybatch_action_cb);
    IupSetCallback(item_match_library, "ACTION", (Icallback) item_matchlibrary_action_cb);
    IupSetCallback(item_search_engine, "ACTION", (Icallback) item_searchengine_action_cb);
    IupSetCallback(item_analysis, "ACTION", (Icallback) toggle_analysis_visibility_cb);
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
    Ihandle *btn_new, *btn_open, *btn_properties;
    Ihandle *btn_copy, *btn_paste;
    Ihandle *btn_prev, *btn_next;
    Ihandle *btn_edit, *btn_analysis, *btn_search;
    Ihandle *btn_blunder, *btn_dice, *btn_cube, *btn_score;
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

    btn_properties = IupButton(NULL, NULL);
    IupSetAttribute(btn_properties, "IMAGE", "IUP_FileProperties");
    IupSetAttribute(btn_properties, "FLAT", "Yes");
    IupSetAttribute(btn_properties, "CANFOCUS", "No");
    IupSetAttribute(btn_properties, "TIP", "Database Metadata");

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

    btn_edit = IupButton("Edit", NULL);
    IupSetAttribute(btn_edit, "FLAT", "Yes");
    IupSetAttribute(btn_edit, "CANFOCUS", "No");
    IupSetAttribute(btn_edit, "TIP", "Edit Mode\tTab");

    btn_analysis = IupButton("Analysis", NULL);
    IupSetAttribute(btn_analysis, "FLAT", "Yes");
    IupSetAttribute(btn_analysis, "CANFOCUS", "No");
    IupSetAttribute(btn_analysis, "TIP", "Analysis (Ctrl+L)");

    btn_search = IupButton("Search", NULL);
    IupSetAttribute(btn_search, "FLAT", "Yes");
    IupSetAttribute(btn_search, "CANFOCUS", "No");
    IupSetAttribute(btn_search, "TIP", "Search (Ctrl+F)");

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
            btn_new, btn_open,
            IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
            btn_copy, btn_paste,
            IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
            btn_prev, btn_next,
            IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
            btn_edit, btn_analysis, btn_search,
            IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
            btn_manual,
            NULL);

    IupSetAttribute(ih, "NAME", "TOOLBAR");
    IupSetAttribute(ih, "MARGIN", "5x5");
    IupSetAttribute(ih, "GAP", "2");

    IupSetCallback(btn_new, "ACTION", (Icallback) item_new_action_cb);
    IupSetCallback(btn_open, "ACTION", (Icallback) item_open_action_cb);
    IupSetCallback(btn_properties, "ACTION", (Icallback) item_properties_action_cb);
    IupSetCallback(btn_copy, "ACTION", (Icallback) item_copy_action_cb);
    IupSetCallback(btn_paste, "ACTION", (Icallback) item_paste_action_cb);
    IupSetCallback(btn_next, "ACTION", (Icallback) item_nextposition_action_cb);
    IupSetCallback(btn_prev, "ACTION", (Icallback) item_prevposition_action_cb);
    IupSetCallback(btn_edit, "ACTION", (Icallback) item_editmode_action_cb);
    IupSetCallback(btn_analysis, "ACTION", (Icallback) toggle_analysis_visibility_cb);
    IupSetCallback(btn_search, "ACTION", (Icallback) item_searchengine_action_cb);
    IupSetCallback(btn_preferences, "ACTION", (Icallback) item_preferences_action_cb);
    IupSetCallback(btn_manual, "ACTION", (Icallback) item_helpmanual_action_cb);

    return ih;
}

static Ihandle* create_cmdline(void)
{
    Ihandle* ih;
    Ihandle* formattag;
    ih = IupText(NULL);
    IupSetAttribute(ih, "NAME", "CMDLINE");
    IupSetAttribute(ih, "EXPAND", "HORIZONTAL");
    IupSetAttribute(ih, "BORDER", "YES");
    IupSetAttribute(ih, "SIZE", "x10");
    IupSetAttribute(ih, "FONTSIZE", "12");
    IupSetAttribute(ih, "CANFOCUS", "YES");
    IupSetAttribute(ih, "CHANGECASE", "UPPER");
    /* IupSetAttribute(ih, "NC", "82"); */

    IupSetAttribute(ih, "VISIBLE", "NO");
    IupSetAttribute(ih, "FLOATING", "YES");
    return ih;
}

static Ihandle* create_statusbar(void)
{
    Ihandle *ih;

    char text[100];
    text[0] = '\0';
    strcat(text, mode_to_str(mode_active));
    sb_mode = IupLabel(text);
    IupSetAttribute(sb_mode, "NAME", "SB_MODE");
    IupSetAttribute(sb_mode, "CANFOCUS", "NO");
    IupSetAttribute(sb_mode, "FONTSIZE", SB_DEFAULT_FONTSIZE);

    sb_lib = IupLabel(msg_info_no_position);
    IupSetAttribute(sb_lib, "NAME", "SB_LIB");
    IupSetAttribute(sb_lib, "CANFOCUS", "NO");
    /* IupSetAttribute(sb_mode, "SIZE", "40x10"); */
    IupSetAttribute(sb_lib, "FONTSIZE", SB_DEFAULT_FONTSIZE);

    sb_msg = IupLabel("MSG INFO");
    IupSetAttribute(sb_msg, "NAME", "SB_MSG");
    IupSetAttribute(sb_msg, "EXPAND", "HORIZONTAL");
    IupSetAttribute(sb_msg, "CANFOCUS", "NO");
    IupSetAttribute(sb_msg, "FONTSIZE", SB_DEFAULT_FONTSIZE);

    ih = IupHbox(sb_mode,
            IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
            sb_lib,
            IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
            sb_msg,
            IupSetAttributes(IupLabel(NULL), "SEPARATOR=VERTICAL"),
            NULL);
    IupSetAttribute(ih, "NAME", "STATUSBAR");
    IupSetAttribute(ih, "EXPAND", "HORIZONTAL");
    IupSetAttribute(ih, "PADDIND", "10x5");

    return ih;
}

static int update_sb_mode(){
    IupSetAttribute(sb_mode, "TITLE", mode_to_str(mode_active));
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int update_sb_msg(const char* msg_new){
    IupSetAttribute(sb_msg, "TITLE", msg_new);
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int update_sb_lib(){
    sprintf(_c, "%s : %i/%i", lib_list[lib_index],
            pos_index+1, pos_nb);
    IupSetAttribute(sb_lib, "TITLE", _c);
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int switch_to_library(const char* l, int* lib_index){
    if(strcmp(l,"main")==0){
        *lib_index=LIBRARIES_NUMBER_MAX-1;
    } else if(strcmp(l,"mix")==0){
        *lib_index=LIBRARIES_NUMBER_MAX-1;
    } else {
        db_get_library_id_from_name(db,l,lib_index);
    }
    update_sb_lib();
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

    ih=IupText(NULL);
    IupSetAttribute(ih, "NAME", "CHECKER_ANALYSIS");
    IupSetAttribute(ih, "EXPAND", "YES");
    IupSetAttribute(ih, "BORDER", "NO");
    IupSetAttribute(ih, "FONTSIZE", "12");
    IupSetAttribute(ih, "FONT", "Nimbus Mono PS, 10");
    IupSetAttribute(ih, "MULTILINE", "YES");
    IupSetAttribute(ih, "ALIGNMENT", "ALEFT");
    IupSetAttribute(ih, "READONLY", "YES");
    IupSetAttribute(ih, "AUTOHIDE", "YES");

    IupSetAttribute(ih, "VISIBLE", "NO");
    IupSetAttribute(ih, "FLOATING", "YES");
    return ih;
}

static Ihandle* create_search(void)
{
    Ihandle *ih;
    ih = IupCells();
    IupSetAttribute(ih, "BOXED", "YES");

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

int add_position_to_library(sqlite3* db, const int pos_id,
        const char* l){
    if(!db_is_valid_library_name(l)){
        update_sb_msg(msg_err_invalid_library_name);
        return 0;
    }
    if(db_library_exists(db, l)){
        printf("library already exists!\n");
        printf("pos_id lib_name: %i %s\n", pos_id, l);
        if(!db_is_position_in_library(db,pos_id,l)){
            printf("position is not in library\n");
            db_insert_position_to_library(db,pos_id,l);
            update_sb_msg(msg_info_position_added_to_library);
        }
    } else {
        printf("library does not exists\n");
        db_insert_library(db, l);
        db_select_libraries(db, &lib_nb, lib_list_id,
                lib_list);
        db_insert_position_to_library(db,pos_id,l);
        update_sb_msg(msg_info_position_added_to_library);
    }
    return 1;
}

int delete_position_from_library(sqlite3* db, const int pos_id,
        const char* l){
    if(!db_is_valid_library_name(l)){
        update_sb_msg(msg_err_invalid_library_name);
        return 0;
    }
    if(db_library_exists(db, l)){
        printf("library exists\n");
        printf("pos_id lib_name: %i %s\n", pos_id, l);
        if(db_is_position_in_library(db,pos_id,l)){
            printf("position is in library\n");
            db_remove_position_from_library(db,pos_id,l);
            update_sb_msg(msg_info_position_removed_from_library);
        }
        db_delete_library_if_void(db,l);
        db_select_libraries(db, &lib_nb, lib_list_id, lib_list);
        db_select_position(db, &pos_nb, pos_list_id, pos_list);
        goto_last_position_cb();
        lib_index=LIBRARIES_NUMBER_MAX-1; //main lib
        update_sb_lib();
    }
    return 1;
}


int parse_cmdline(char* cmdtext){
    printf("\nparse_cmdline\n");
    token_nb=0;
    char *c = strtok(cmdtext, " ");
    while(c!=NULL){
        cmdtoken[token_nb]=c;
        token_nb+=1;
        c=strtok(NULL, " ");
    }
    for(int i=0;i<token_nb;i++){
        printf("token %i: %s\n",i,cmdtoken[i]);
    }

    if(strncmp(cmdtoken[0], ":o", 2)==0){
        printf("\n:o\n");
        item_open_action_cb();
    } else if(strncmp(cmdtoken[0], ":n", 2)==0){
        printf("\n:n\n");
        item_new_action_cb();
    } else if(strncmp(cmdtoken[0], ":q", 2)==0){
        printf("\n:q\n");
        item_exit_action_cb();
    }

    if(db==NULL) {
        update_sb_msg(msg_err_no_db_opened);
        return 0;
    }
    if(strncmp(cmdtoken[0], ":ls", 3)==0){
        db_select_libraries(db, &lib_nb, lib_list_id,
                lib_list);
        char msg_lib[10000], t[100]; msg_lib[0]='\0'; t[0]='\0';
        sprintf(msg_lib, "Librairies: ");
        for(int i=0;i<lib_nb;i++){
            sprintf(t, "%s ", lib_list[i]);
            strcat(msg_lib, t);
        }
        update_sb_msg(msg_lib);
        for(int i=0;i<lib_nb;i++) printf("lib %i %s\n",
                lib_list_id[i], lib_list[i]);
    } else if(strncmp(cmdtoken[0], ":LS", 3)==0){
        printf("\n:LS\n");
        char lname_list[LIBRARIES_NUMBER_MAX][LIBRARY_NAME_MAX];
        int lname_nb;
        db_get_libraries_related_to_position(db,
                pos_list_id[pos_index], &lname_nb, lname_list);
        char msg_lib[10000]; char t[100]; msg_lib[0]='\0';t[0]='\0';
        strcat(msg_lib,"This position belongs to:");
        for(int i=0;i<lname_nb;i++){
            sprintf(t," %s",lname_list[i]);
            strcat(msg_lib,t);
        }
        update_sb_msg(msg_lib);
    } else if(strncmp(cmdtoken[0], ":d", 2)==0){
        printf("\n:d\n");
        char *lname;
        if(token_nb==1){
            printf("token_nb lib_index lib_list: %i %i %s\n",token_nb,lib_index,lib_list[lib_index]);
            lname=lib_list[lib_index];
            db_delete_library(db,lname);
            char t[100]; t[0]='\0'; sprintf(t, "%s has been deleted.",lname);
            update_sb_msg(t);
        } else {
            char t[100], t0[100]; t[0]='\0'; t0[0]='\0';
            for(int i=1;i<token_nb;i++){
                lname=cmdtoken[i];
                printf("lname %s\n",lname);
                db_delete_library(db,lname);
                sprintf(t0, "%s ",lname);
                strcat(t,t0);
            }
            if(token_nb==2) strcat(t,"has been deleted.");
            if(token_nb>2) strcat(t,"have been deleted.");
            update_sb_msg(t);
        }
        db_select_libraries(db, &lib_nb, lib_list_id, lib_list);
        lib_index=LIBRARIES_NUMBER_MAX-1; //main lib
        update_sb_lib();
        goto_first_position_cb();
    } else if(strncmp(cmdtoken[0], ":i", 2)==0){
        printf("\n:i\n");
        item_import_action_cb();
    } else if(strncmp(cmdtoken[0], ":mv", 3)==0){
        printf("\n:mv\n");
        if(token_nb==1 || token_nb>3) return 1; //invalid syntax
        char *lname_old, *lname_new;
        if(token_nb==2){ //rename current lib
            if(lib_index==LIBRARIES_NUMBER_MAX-1 ||
                    lib_index==LIBRARIES_NUMBER_MAX-2) return 1; //main/mix, nothing to do
            lname_old=lib_list[lib_index];
            lname_new=cmdtoken[1];
            db_rename_library(db,lname_old,lname_new);
        } else if(token_nb==3){
            printf("token_nb old new: %i  %s %s\n",token_nb, lname_old, lname_new);
            lname_old=cmdtoken[1];
            lname_new=cmdtoken[2];
            if(!db_library_exists(db,lname_old)){
                char t[100]; t[0]='\0'; sprintf(t, "%s does not exists.",lname_old);
                update_sb_msg(t);
                return 1;
            }
            db_rename_library(db,lname_old,lname_new);
        }
        char t[100]; t[0]='\0'; sprintf(t, "%s has been renamed to %s.",lname_old,lname_new);
        update_sb_msg(t);
        db_select_libraries(db, &lib_nb, lib_list_id, lib_list);
        update_sb_lib();
    } else if(strncmp(cmdtoken[0], ":cp", 3)==0){
        printf("\n:cp\n");
        char *lname_old, *lname_new;
        if(token_nb==1) return 1; //invalid syntax
        if(token_nb==2){
            lname_old=lib_list[lib_index];
            lname_new=cmdtoken[1];
            db_copy_library(db,lname_old,lname_new);
            char t[100]; t[0]='\0'; sprintf(t, "%s has been duplicated, New copy: %s.",lname_old,lname_new);
            update_sb_msg(t);
        } else if(token_nb>=3){
            lname_old=cmdtoken[1];
            char t[1000], t0[1000]; t[0]='\0'; t0[0]='\0';
            if(token_nb==3) sprintf(t, "%s has been duplicated. New copy:",lname_old);
            if(token_nb>3) sprintf(t, "%s has been duplicated. New copies:",lname_old);
            for(int i=2;i<token_nb;i++){
                lname_new=cmdtoken[i];
                db_copy_library(db,lname_old,lname_new);
                sprintf(t0," %s",lname_new);
                strcat(t,t0);
                update_sb_msg(t);
            }
            strcat(t, ".");
        }
        db_select_libraries(db, &lib_nb, lib_list_id, lib_list);
        update_sb_lib();
    } else if(strncmp(cmdtoken[0], ":w!", 3)==0){
        printf(":w!\n");
        bool exist=false;
        int nb=0;
        int _id[1000];
        db_find_identical_position(db, pos_ptr, &exist, &nb, _id);
        if(exist){
            goto_position_cb(&_id[0]);
            update_sb_msg(msg_info_position_already_exists);
            printf("Position already exists. nb: %i\n", nb);
            for(int i=0;i<nb;i++) printf("_id[%i]: %i\n",i, _id[i]);
        } else {
            int id=pos_list_id[pos_index];
            if(db_analysis_exist(db,id)){
                goto_position_cb(&id);
                mode_active=NORMAL;
                update_sb_msg(msg_err_cannot_update_position_with_analysis);
                update_sb_mode();
            } else {
                db_update_position(db, &id, pos_ptr);
                db_select_position(db, &pos_nb,
                        pos_list_id, pos_list);
                mode_active=NORMAL;
                update_sb_msg(msg_info_position_updated);
                update_sb_mode();
            }
        }
        if(token_nb>1){
            int pos_id = pos_list_id[pos_index];
            for(int i=1;i<token_nb;i++){
                char *l=cmdtoken[i];
                add_position_to_library(db,pos_id,l); 
            }
        }

    } else if(strncmp(cmdtoken[0], ":w", 2)==0){
        printf(":w\n");
        bool exist=false;
        int nb=0;
        int _id[1000];
        db_find_identical_position(db, pos_ptr, &exist, &nb, _id);
        if(exist){
            goto_position_cb(&_id[0]);
            update_sb_msg(msg_info_position_already_exists);
            printf("Position already exists. nb: %i\n", nb);
            for(int i=0;i<nb;i++) printf("_id: %i\n", _id[i]);
        } else {
            db_insert_position(db, pos_ptr);
            update_sb_msg(msg_info_position_written);
            db_select_position(db, &pos_nb,
                    pos_list_id, pos_list);
            goto_last_position_cb();
        }
        if(token_nb>1){
            char t[10000], t0[1000]; t[0]='\0'; t0[0]='\0';
            strcat(t, "The position has been");
            int pos_id = pos_list_id[pos_index];
            for(int i=1;i<token_nb;i++){
                char *l=cmdtoken[i];
                if(l[0]!='-'){
                    add_position_to_library(db,pos_id,l); 
                    sprintf(t0," added to %s", l);
                } else if(strcmp(l,"-")==0
                        && lib_index!=LIBRARIES_NUMBER_MAX-1
                        && lib_index!=LIBRARIES_NUMBER_MAX-2){
                    char l2[LIBRARY_NAME_MAX]; l2[0]='\0';
                    strcat(l2, lib_list[lib_index]);
                    delete_position_from_library(db,pos_id,l2);
                    sprintf(t0," removed from %s",l2);
                } else {
                    l++;
                    delete_position_from_library(db,pos_id,l);
                    sprintf(t0," removed from %s",l);
                }
                strcat(t,t0);
                if(i==token_nb-1) {strcat(t,".");} else {strcat(t,",");} 
            }
            update_sb_msg(t);
        }
    } else if(strncmp(cmdtoken[0], ":e", 2)==0){
        printf(":e\n");
        printf("token_nb %i\n",token_nb);
        if(token_nb>1){
            db_select_position_from_libraries(db, cmdtoken, token_nb,
                    &pos_nb, pos_list_id, pos_list);
            if(token_nb==2){ //update display if only specific lib
                char *l; l=cmdtoken[1]; int l_id;
                if(db_library_exists(db,l)){
                    db_get_library_id_from_name(db,l,&l_id);
                    lib_index=find_index_from_int(l_id, lib_list_id, lib_nb);
                    printf("lib_nb %i\n",lib_nb);
                    for(int i=0;i<lib_nb;i++) printf("i lib %i %s\n",i,lib_list[i]);
                    printf("lib_index lib_list %i %s\n",lib_index, lib_list[lib_index]);
                    update_sb_lib();
                    char t[100]; t[0]='\0'; sprintf(t, "Switched to %s.",lib_list[lib_index]);
                    update_sb_msg(t);
                } else {
                    db_select_position(db, &pos_nb, pos_list_id, pos_list);
                    lib_index=LIBRARIES_NUMBER_MAX-1; //main lib
                    update_sb_lib();
                    char t[100]; t[0]='\0'; sprintf(t, "Switched to %s.",lib_list[lib_index]);
                }
            } else {
                lib_index=LIBRARIES_NUMBER_MAX-2; //mix lib
                update_sb_lib();
                char t[100]; t[0]='\0'; sprintf(t, "Switched to %s.",lib_list[lib_index]);
            }
        } else {
            db_select_position(db, &pos_nb, pos_list_id, pos_list);
            lib_index=LIBRARIES_NUMBER_MAX-1; //main lib
            update_sb_lib();
            char t[100]; t[0]='\0'; sprintf(t, "Switched to %s.",lib_list[lib_index]);
            update_sb_msg(t);
        }
        if(pos_nb==0){
            pos_list[0]=POS_DEFAULT;
            pos_list_id[0]=-1;
            pos_nb=1;
        }
        goto_first_position_cb();
    } else if(strncmp(cmdtoken[0], ":D", 2)==0){
        printf("\n:D\n");
        int id = pos_list_id[pos_index];
        db_remove_position_from_libraries(db,id);
        db_delete_position(db, &id);
        update_sb_msg(msg_info_position_deleted);
    } else if(strncmp(cmdtoken[0], ":s", 2)==0){
        printf(":s\n");
        int force_cube=0;
        int force_score=0;
        int criteria_blunder=0;
        int criteria_pipcount=0;
        int criteria_abspipcount=0;
        int criteria_checkeroff=0;
        int criteria_backchecker1=0;
        int criteria_backchecker2=0;
        int criteria_zone1=0;
        int criteria_zone2=0;
        int criteria_equity=0;
        int bmin=0, bmax=0;
        int pmin=0, pmax=0;
        int Pmin=0, Pmax=0;
        int omin=0;
        int bc_num1, bc_num2; //backchecker
        int z_num1, z_num2; //zone
        int emin=0, emax=0;
        for(int i=1;i<token_nb;i++){
            printf("tok %i %s\n",i,cmdtoken[i]); 
            if(strncmp(cmdtoken[i],"c",1)==0
                    || strncmp(cmdtoken[i],"cu",2)==0
                    || strncmp(cmdtoken[i],"cube",4)==0){
                force_cube=1;
            } else if(strncmp(cmdtoken[i],"s",1)==0
                    || strncmp(cmdtoken[i],"sc",2)==0
                    || strncmp(cmdtoken[i],"score",5)==0){
                force_score=1;
            } else if(strncmp(cmdtoken[i],"o",1)==0){
                sscanf(cmdtoken[i], "o%d", &omin);
                criteria_checkeroff=1;
                printf("\ncriteria checkeroff: %i\n", omin);
            } else if(strncmp(cmdtoken[i],"P",1)==0){
                sscanf(cmdtoken[i], "P%d,%d", &pmin, &pmax);
                if(pmax<pmin) int_swap(&pmax, &pmin);
                criteria_pipcount=1;
                printf("\ncriteria pipcount: %i %i\n", pmin, pmax);
            } else if(strncmp(cmdtoken[i],"p",1)==0){
                sscanf(cmdtoken[i], "p%d,%d", &Pmin, &Pmax);
                if(Pmax<Pmin) int_swap(&Pmax, &Pmin);
                criteria_abspipcount=1;
                printf("\ncriteria absolut pipcount: %i %i\n", Pmin, Pmax);
            } else if(strncmp(cmdtoken[i],"k",1)==0){
                sscanf(cmdtoken[i], "k%d", &bc_num1);
                criteria_backchecker1=1;
                printf("\ncriteria backchecker 1: %i\n", bc_num1);
            } else if(strncmp(cmdtoken[i],"K",1)==0){
                sscanf(cmdtoken[i], "K%d", &bc_num2);
                criteria_backchecker2=1;
                printf("\ncriteria backchecker : %i\n", bc_num2);
            } else if(strncmp(cmdtoken[i],"z",1)==0){
                sscanf(cmdtoken[i], "z%d", &z_num1);
                criteria_zone1=1;
                printf("\ncriteria zone 1: %i\n", z_num1);
            } else if(strncmp(cmdtoken[i],"Z",1)==0){
                sscanf(cmdtoken[i], "Z%d", &z_num2);
                criteria_zone2=1;
                printf("\ncriteria zone : %i\n", z_num2);
            } else if(strncmp(cmdtoken[i],"b",1)==0){
                if(strncmp(cmdtoken[i],"b<",2)==0){
                    criteria_blunder=1;
                    sscanf(cmdtoken[i], "b<%d", &bmax);
                    bmin=-1;
                    printf("\ncriteria blunder max: %d\n",bmax);
                }else if(strncmp(cmdtoken[i],"b>",2)==0){
                    criteria_blunder=1;
                    sscanf(cmdtoken[i], "b>%d", &bmin);
                    bmax=10000;
                    printf("\ncriteria blunder min: %d\n",bmin);
                }else if(strstr(cmdtoken[i],",")!=0){
                    criteria_blunder=1;
                    sscanf(cmdtoken[i], "b%d,%d", &bmin, &bmax);
                    if(bmax<bmin) int_swap(&bmax, &bmin);
                    printf("\ncriteria blunder min max: %d %d\n",bmin,bmax);
                }
            } else if(strncmp(cmdtoken[i],"e",1)==0){
                if(strncmp(cmdtoken[i],"e<",2)==0){
                    criteria_equity=1;
                    sscanf(cmdtoken[i], "e<%d", &emax);
                    emin=-10000;
                    printf("\ncriteria equity max: %d\n",emax);
                }else if(strncmp(cmdtoken[i],"e>",2)==0){
                    criteria_equity=1;
                    sscanf(cmdtoken[i], "e>%d", &emin);
                    emax=10000;
                    printf("\ncriteria equity min: %d\n",emin);
                }else if(strstr(cmdtoken[i],",")!=0){
                    criteria_equity=1;
                    sscanf(cmdtoken[i], "e%d,%d", &emin, &emax);
                    if(emax<emin) int_swap(&emax, &emin);
                    printf("\ncriteria equity min max: %d %d\n",emin,emax);
                }
            }
        }
        db_select_specific_position(db, pos_ptr,
                force_cube, force_score,
                criteria_blunder, bmin, bmax,
                criteria_equity, emin, emax,
                &pos_nb, pos_list_id, pos_list);
        if(criteria_checkeroff) filter_position_by_checkeroff(omin);
        if(criteria_pipcount) filter_position_by_pipcount(pmin,pmax,0);
        if(criteria_abspipcount) filter_position_by_pipcount(Pmin,Pmax,true);
        if(criteria_backchecker1) filter_position_by_backchecker(PLAYER1,bc_num1);
        if(criteria_backchecker2) filter_position_by_backchecker(PLAYER2,bc_num2);
        if(criteria_zone1) filter_position_by_checker_in_the_zone(PLAYER1,z_num1);
        if(criteria_zone2) filter_position_by_checker_in_the_zone(PLAYER2,z_num2);
        if(pos_nb==0){
            pos_list[0]=POS_DEFAULT;
            pos_list_id[0]=-1;
            pos_nb=1;
        }
        goto_first_position_cb();
    }
    return 1;
}

int update_checker_analysis(const int pid){
    printf("\nupdate_checker_analysis\n");
    char txt[ANALYSIS_MULTILINE_MAX]; txt[0]='\0';
    char t[1000]; t[0]='\0';
    char t1[1000]; t1[0]='\0';
    db_select_checker_analysis(db,pid,ca,&ca_nb);
    printf("ca_nb: %i\n", ca_nb);
    for(int i=0;i<ca_nb;i++){
        sprintf(t1,"%s %s %s %s",
                ca[i].move1,ca[i].move2,ca[i].move3,ca[i].move4);
        sprintf(t,"%-18s  %+6.3f %+6.3f   ",t1,ca[i].equity,ca[i].error);
        strcat(txt,t);
        sprintf(t,"P: %5.2f %5.2f %5.2f  O: %5.2f %5.2f %5.2f   %-11s\n",
                ca[i].p1_w,ca[i].p1_g,ca[i].p1_b,
                ca[i].p2_w,ca[i].p2_g,ca[i].p2_b,
                ca[i].depth);
        strcat(txt,t);
    }
    printf("txt: %s\n", txt);
    IupSetAttribute(analysis, "VALUE", txt);
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

int update_cube_analysis(const int pid){
    printf("\nupdate_cube_analysis\n");
    char txt[ANALYSIS_MULTILINE_MAX]; txt[0]='\0';
    char t[1000]; t[0]='\0';
    /* char t1[1000]; t1[0]='\0'; */
    db_select_cube_analysis(db,pid,&da,&da_nb);
    printf("da_nb: %i\n", da_nb);
    if(da_nb==0){
        IupSetAttribute(analysis, "VALUE", txt);
        IupRefresh(dlg);
        return IUP_DEFAULT;
    }
    sprintf(t,"P: %5.2f  %5.2f  %5.2f  O: %5.2f  %5.2f  %5.2f\n",
            da.p1_w,da.p1_g,da.p1_b,
            da.p2_w,da.p2_g,da.p2_b
            );
    strcat(txt,t);
    if(pos_ptr->cube==0){
    sprintf(t,"Cubeless Equities:     No Double: %+6.3f    Double: %+6.3f\n",
            da.cubeless_equity_nd, da.cubeless_equity_d); 
    }else{
    sprintf(t,"Cubeless Equities:   No Redouble: %+6.3f    Double: %+6.3f\n",
            da.cubeless_equity_nd, da.cubeless_equity_d); 
    }
    strcat(txt,t);
    sprintf(t,"%17s   %15s    %+6.3f    %+6.3f\n",
            "Cubeful Equities:","No double",da.cubeful_equity_nd,da.error_nd);
    strcat(txt,t);
    if(pos_ptr->cube==0){
        sprintf(t,"%17s   %15s    %+6.3f    %+6.3f\n",
                "","Double/Take",da.cubeful_equity_dt,da.error_dt);
        strcat(txt,t);
        sprintf(t,"%17s   %15s    %+6.3f    %+6.3f\n",
                "","Double/Pass",da.cubeful_equity_dp,da.error_dp);
        strcat(txt,t);
    }else{
        if(da.beaver){
            sprintf(t,"%17s   %15s    %+6.3f    %+6.3f\n",
                    "","Redouble/Beaver",da.cubeful_equity_dt,da.error_dt);
        }else{
            sprintf(t,"%17s   %15s    %+6.3f    %+6.3f\n",
                    "","Redouble/Take",da.cubeful_equity_dt,da.error_dt);
        }
        strcat(txt,t);
        sprintf(t,"%17s   %15s    %+6.3f    %+6.3f\n",
                "","Redouble/Pass",da.cubeful_equity_dp,da.error_dp);
        strcat(txt,t);
    }
    sprintf(t,"Depth analysis: %s\n",da.depth);
    strcat(txt,t);

    printf("txt: %s\n", txt);
    IupSetAttribute(analysis, "VALUE", txt);
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

/* END Interface */

/************************ Drawing *************************/

/* BEGIN Drawing */

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
#define DICE_SIZE (POINT_SIZE)
#define DICE_POINTSIZE 2
#define DICE_XPOS (BOARD_WIDTH/2. +1.3*POINT_SIZE)
#define DICE_GAP (0.2*POINT_SIZE)
#define DICE1_XPOS (DICE_XPOS-0.5*DICE_SIZE-0.5*DICE_GAP)
#define DICE2_XPOS (DICE_XPOS+0.5*DICE_SIZE+0.5*DICE_GAP)
#define DICE_ROLL_YPOS_UP (0.25*BOARD_HEIGHT/2.)
#define DICE_ROLL_YPOS_DOWN (-DICE_ROLL_YPOS_UP)
#define DICE_CUBE_YPOS_UP (0.75*BOARD_HEIGHT/2.)
#define DICE_CUBE_YPOS_DOWN (-DICE_CUBE_YPOS_UP)
#define DICE_ROTATION_ANGLE 20.
#define DICE_LINECOLOR CD_BLACK
#define DICE_LINEWIDTH 1
#define DICE1_COLOR CD_BLACK
#define DICE2_COLOR CD_BLUE
#define DICE1_POINT_COLOR CD_BLUE
#define DICE2_POINT_COLOR CD_BLACK
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

int board_direction = BOARD_DIRECTION;

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

void draw_dice(cdCanvas *cv, const int *dice, const int player_on_roll,
        const int cube_action){
    double dice1_x, dice2_x;

    void draw_singledie(cdCanvas *cv, const int die,
            const double x, const double y){
        cdCanvasForeground(cv, DICE_LINECOLOR);
        cdCanvasLineStyle(cv, CD_CONTINUOUS);
        wdCanvasLineWidth(cv, DICE_LINEWIDTH);
        cdCanvasMarkType(cv, CD_CIRCLE);
        wdCanvasMarkSize(cv, DICE_POINTSIZE);
        wdCanvasRect(cv,x-0.5*DICE_SIZE,x+0.5*DICE_SIZE,
                y-0.5*DICE_SIZE,y+0.5*DICE_SIZE);
        if(pos_ptr->cube_action==1) return;
        switch(die){
            case 1:
                wdCanvasMark(cv,x,y);
                break;
            case 2:
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                break;
            case 3:
                wdCanvasMark(cv,x,y);
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                break;
            case 4:
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                break;
            case 5:
                wdCanvasMark(cv,x,y);
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                break;
            case 6:
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y);
                wdCanvasMark(cv,x-0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y-0.3*DICE_SIZE);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y);
                wdCanvasMark(cv,x+0.3*DICE_SIZE,y+0.3*DICE_SIZE);
                break;
            default:
                break;
        }
    }

    //la couleur du dé est fonction du joueur
    dice1_x = DICE1_XPOS;
    dice2_x = DICE2_XPOS;
    if(cube_action==1){
        if(player_on_roll==PLAYER1){
            draw_singledie(cv,dice[0],dice1_x, DICE_CUBE_YPOS_DOWN);
            draw_singledie(cv,dice[1],dice2_x, DICE_CUBE_YPOS_DOWN);
        } else if(player_on_roll==PLAYER2){
            draw_singledie(cv,dice[0],dice1_x, DICE_CUBE_YPOS_UP);
            draw_singledie(cv,dice[1],dice2_x, DICE_CUBE_YPOS_UP);
        }
    } else{
        if(player_on_roll==PLAYER1){
            draw_singledie(cv,dice[0],dice1_x, DICE_ROLL_YPOS_DOWN);
            draw_singledie(cv,dice[1],dice2_x, DICE_ROLL_YPOS_DOWN);
        } else if(player_on_roll==PLAYER2){
            draw_singledie(cv,dice[0],dice1_x, DICE_ROLL_YPOS_UP);
            draw_singledie(cv,dice[1],dice2_x, DICE_ROLL_YPOS_UP);
        }
    }
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

void draw_pointnumber(cdCanvas* cv, const int orientation,
        const int p_on_roll) {
    double x, y;
    char t[3];
    cdCanvasForeground(cv, POINTNUMBER_LINECOLOR);
    cdCanvasTextAlignment(cv, CD_CENTER);
    cdCanvasFont(cv, POINTNUMBER_FONT, POINTNUMBER_STYLE, POINTNUMBER_FONTSIZE);
    if(orientation>0) {

        x = BOARD_WIDTH/2 -POINT_SIZE/2;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=1; i<7; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -POINT_SIZE;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=7; i<13; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -BOARD_WIDTH/2 +POINT_SIZE/2;
        y = POINTNUMBER_YPOS_UP;
        for(int i=13; i<19; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = POINT_SIZE;
        y = POINTNUMBER_YPOS_UP;
        for(int i=19; i<25; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

    } else {

        x = -BOARD_WIDTH/2 +POINT_SIZE/2;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=1; i<7; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = POINT_SIZE;
        y = POINTNUMBER_YPOS_DOWN;
        for(int i=7; i<13; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
            wdCanvasText(cv, x, y, t);
            x += POINT_SIZE;
        }

        x = BOARD_WIDTH/2 -POINT_SIZE/2;
        y = POINTNUMBER_YPOS_UP;
        for(int i=13; i<19; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
            wdCanvasText(cv, x, y, t);
            x -= POINT_SIZE;
        }

        x = -POINT_SIZE;
        y = POINTNUMBER_YPOS_UP;
        for(int i=19; i<25; i++){
            if(p_on_roll==PLAYER2){ sprintf(t, "%d", 25-i);
            }else{ sprintf(t, "%d", i); }
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

    if(board_direction==1) eps = 1;
    if(board_direction!=1) eps = -1;

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

    if(db==NULL){
        update_sb_msg(msg_info_no_db_loaded);
        return;
    }

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
    draw_checker(cv, pos_ptr, board_direction);
    draw_cube(cv, pos_ptr->cube);
    draw_dice(cv,pos_ptr->dice,pos_ptr->player_on_roll,
            pos_ptr->cube_action);
    draw_checkeroff(cv, off1, PLAYER1, board_direction);
    draw_checkeroff(cv, off2, PLAYER2, board_direction);
    if(is_pointletter_active) {
        draw_pointletter(cv, board_direction, pos_ptr->cube);
    } else {
        draw_pointnumber(cv, board_direction,
                pos_ptr->player_on_roll);
    }
    draw_score(cv, pos_ptr->p1_score, PLAYER1);
    draw_score(cv, pos_ptr->p2_score, PLAYER2);
    draw_pipcount(cv, pip1, PLAYER1);
    draw_pipcount(cv, pip2, PLAYER2);

    cdCanvasFlush(cv);
}

/* END Drawing */


/*************** Keyboard Shortcuts ***********************/
/* BEGIN Keyboard Shortcuts */

static void set_keyboard_shortcuts()
{

    IupSetCallback(dlg, "K_TAB", (Icallback) toggle_editmode_cb);
    IupSetCallback(dlg, "K_ESC", (Icallback) esc_cb);
    IupSetCallback(dlg, "K_minus", (Icallback) minus_cb);
    IupSetCallback(dlg, "K_bracketleft", (Icallback) bracketleft_cb);
    IupSetCallback(dlg, "K_bracketright", (Icallback) bracketright_cb);
    IupSetCallback(dlg, "K_CR", (Icallback) cr_cb);
    IupSetCallback(dlg, "K_BS", (Icallback) backspace_cb);
    IupSetCallback(dlg, "K_SP", (Icallback) space_cb);
    IupSetCallback(dlg, "K_LEFT", (Icallback) left_cb);
    IupSetCallback(dlg, "K_RIGHT", (Icallback) right_cb);
    IupSetCallback(dlg, "K_PGUP", (Icallback) pgup_cb);
    IupSetCallback(dlg, "K_PGDN", (Icallback) pgdn_cb);
    IupSetCallback(dlg, "K_cLEFT", (Icallback) board_direction_left_cb);
    IupSetCallback(dlg, "K_cRIGHT", (Icallback) board_direction_right_cb);
    IupSetCallback(dlg, "K_cUP", (Icallback) display_player_on_roll_up);
    IupSetCallback(dlg, "K_cDOWN", (Icallback) display_player_on_roll_down);


    IupSetCallback(dlg, "K_cN", (Icallback) item_new_action_cb);
    IupSetCallback(dlg, "K_cO", (Icallback) item_open_action_cb);
    IupSetCallback(dlg, "K_cS", (Icallback) item_save_action_cb);
    IupSetCallback(dlg, "K_cQ", (Icallback) item_exit_action_cb);
    IupSetCallback(dlg, "K_cI", (Icallback) item_import_action_cb);
    IupSetCallback(dlg, "K_cL", (Icallback) toggle_analysis_visibility_cb);
    IupSetCallback(dlg, "K_cV", (Icallback) item_paste_action_cb);
    IupSetCallback(dlg, "K_cC", (Icallback) item_copy_action_cb);
    IupSetCallback(dlg, "K_cF", (Icallback) item_searchengine_action_cb);

    IupSetCallback(dlg, "K_a", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_b", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_c", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_d", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_e", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_f", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_g", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_h", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_i", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_j", (Icallback) j_cb);
    IupSetCallback(dlg, "K_k", (Icallback) k_cb);
    IupSetCallback(dlg, "K_l", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_m", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_n", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_o", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_p", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_q", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_r", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_s", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_t", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_u", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_v", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_w", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_x", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_y", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_z", (Icallback) editmode_letter_cb);

    IupSetCallback(dlg, "K_A", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_B", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_C", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_D", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_E", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_F", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_G", (Icallback) G_cb);
    IupSetCallback(dlg, "K_H", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_I", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_J", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_K", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_L", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_M", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_N", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_O", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_P", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_Q", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_R", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_S", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_T", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_U", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_V", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_W", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_X", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_Y", (Icallback) editmode_letter_cb);
    IupSetCallback(dlg, "K_Z", (Icallback) editmode_letter_cb);

    IupSetCallback(dlg, "K_1", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_2", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_3", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_4", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_5", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_6", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_7", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_8", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_9", (Icallback) editmode_digit_cb);
    IupSetCallback(dlg, "K_0", (Icallback) editmode_digit_cb);

}

/* END Keyboard Shortcuts */

/************************ Callbacks ***********************/
// BEGIN Callbacks

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

static int canvas_dropfiles_cb(Ihandle* ih, const char* filename,
        int num, int x, int y)
{
    printf("\ncanvas_dropfiles_cb\n");
    printf("filename: %s\n", filename);
    if(has_extension(filename, "db")){
        int result = db_open(filename);
        if (result != 0) {
            update_sb_msg(msg_err_failed_to_open_db);
            printf("%s\n",msg_err_failed_to_open_db);
            return result;
        }
        db_select_position(db, &pos_nb,
                pos_list_id, pos_list);
        db_select_libraries(db, &lib_nb, lib_list_id, lib_list);
        goto_first_position_cb();
        update_sb_lib();
        update_sb_msg(msg_info_db_loaded);
        printf("%s\n",msg_info_db_loaded);

    } else if(has_extension(filename, "txt")){
        if (db==NULL) {
            update_sb_msg(msg_err_no_db_opened);
            printf("%s\n",msg_err_no_db_opened);
            return 0;
        }
        FILE *f=open_input(filename);
        if(f==NULL){
            update_sb_msg(msg_err_failed_to_import_pos);
            printf("%s\n",msg_err_failed_to_import_pos);
        }
        parse_from_file=1;
        txtfiletype_t rc_txtft=check_if_file_is_position_or_match(f); //IMPORTANT ti implement if block
        int pid;
        int rc=db_import_position_from_file(db,f,&pid);
        if(rc){
            db_select_position(db,&pos_nb,pos_list_id,pos_list);
            goto_position_cb(&pid);
            switch_to_library("main",&lib_index);
            update_sb_msg(msg_info_position_imported);
        }else{
            update_sb_msg(msg_err_failed_to_import_pos);
        }
    }
    return IUP_DEFAULT;
}

static int canvas_motion_cb(Ihandle* ih)
{
    /* error_callback(); */
    return IUP_DEFAULT;
}

static int canvas_wheel_cb(Ihandle* ih, float delta,
        int x, int y, char *status)
{
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return 0;
    }
    if(delta>0){
        goto_prev_position_cb();
    } else {
        goto_next_position_cb();
    }
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
    bool is_in_uplabel, is_in_downlabel, is_in_board, is_on_point; 
    bool is_in_cube, is_cube_in_center, is_cube_up, is_cube_down, 
         is_in_cube_positions;
    bool is_on_score1, is_on_score2;
    bool is_in_board2, is_on_bar2, is_in_center2, is_on_point2;
    bool is_in_player1_dice1, is_in_player1_dice2, is_in_player1_roll;
    bool is_in_player2_dice1, is_in_player2_dice2, is_in_player2_roll;
    bool is_in_dice_positions;

    if(mode_active!=EDIT) return IUP_DEFAULT;

    mouse_hold=false;

    if(board_direction==1) dir=1;
    if(board_direction!=1) dir=-1;

    // canvas and world have inverted y axis...
    y2 = cdCanvasInvertYAxis(cdv, y);
    wdCanvasCanvas2World(cdv, x, y2, &xw, &yw);
    ix = round(xw/POINT_SIZE);
    iy = round(yw/POINT_SIZE);
    printf("ix: %i\niy: %i\n", ix, iy);

    // labels (number or point) are in the board
    is_in_board = abs(ix)<=6 && abs(iy)<=6;
    is_in_uplabel = is_in_board && iy==6;
    is_in_downlabel = is_in_board && iy==-6;
    is_in_left = ix<0 && ix>=-6;
    is_in_up = iy>0 && iy<=6;
    is_in_down = iy<0 && iy>=-6;
    is_in_right = ix>0 && ix<=6;
    is_on_bar = is_in_board && ix==0;
    is_on_point = is_in_board && ix!=0 && iy!=0;
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
        (yw<=SCORE_YPOS_DOWN+0.5*POINT_SIZE);
    is_on_score2 = (xw>=SCORE_XPOS-1.*POINT_SIZE) &&
        (yw>=SCORE_YPOS_UP-0.5*POINT_SIZE);
    is_in_player1_dice1 = (xw>=DICE1_XPOS-0.5*DICE_SIZE) &&
        (xw<=DICE1_XPOS+0.5*DICE_SIZE) &&
        (yw>=DICE_ROLL_YPOS_DOWN-0.5*DICE_SIZE) &&
        (yw<=DICE_ROLL_YPOS_DOWN+0.5*DICE_SIZE);
    is_in_player1_dice2 = (xw>=DICE2_XPOS-0.5*DICE_SIZE) &&
        (xw<=DICE2_XPOS+0.5*DICE_SIZE) &&
        (yw>=DICE_ROLL_YPOS_DOWN-0.5*DICE_SIZE) &&
        (yw<=DICE_ROLL_YPOS_DOWN+0.5*DICE_SIZE);
    is_in_player2_dice1 = (xw>=DICE1_XPOS-0.5*DICE_SIZE) &&
        (xw<=DICE1_XPOS+0.5*DICE_SIZE) &&
        (yw>=DICE_ROLL_YPOS_UP-0.5*DICE_SIZE) &&
        (yw<=DICE_ROLL_YPOS_UP+0.5*DICE_SIZE);
    is_in_player2_dice2 = (xw>=DICE2_XPOS-0.5*DICE_SIZE) &&
        (xw<=DICE2_XPOS+0.5*DICE_SIZE) &&
        (yw>=DICE_ROLL_YPOS_UP-0.5*DICE_SIZE) &&
        (yw<=DICE_ROLL_YPOS_UP+0.5*DICE_SIZE);
    is_in_player1_roll = (xw>=DICE1_XPOS-0.5*DICE_SIZE) &&
        (xw<=DICE2_XPOS+0.5*DICE_SIZE) &&
        (yw>=DICE_CUBE_YPOS_DOWN-1.0*POINT_SIZE) &&
        (yw<=DICE_CUBE_YPOS_DOWN+0.5*POINT_SIZE);
    is_in_player2_roll = (xw>=DICE1_XPOS-0.5*DICE_SIZE) &&
        (xw<=DICE2_XPOS+0.5*DICE_SIZE) &&
        (yw>=DICE_CUBE_YPOS_UP-1.0*POINT_SIZE) &&
        (yw<=DICE_CUBE_YPOS_UP+0.5*POINT_SIZE);
    is_in_dice_positions = is_in_player1_dice1 ||
        is_in_player1_dice2 || is_in_player1_roll ||
        is_in_player2_dice1 || is_in_player2_dice2 ||
        is_in_player2_roll;

    // for previous mouse state if clicked
    if(mouse.pressed==1) {
        y2p = cdCanvasInvertYAxis(cdv, mouse.y);
        wdCanvasCanvas2World(cdv, mouse.x, y2p, &xwp, &ywp);
        ixp = round(xwp/POINT_SIZE);
        iyp = round(ywp/POINT_SIZE);
        is_in_board2 = abs(ixp)<=6 && abs(iyp)<=6;
        is_on_bar2 = is_in_board2 && ixp==0;
        is_on_point2 = is_in_board2 && ixp!=0 && iyp!=0;
        if((ix!=ixp || iy!=iyp) && is_on_point && is_on_point2) mouse_hold=true;
    }



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
                if(board_direction==1) i=19+ix;
                if(board_direction!=1) i=18-ix;
            } else if(is_in_down) {
                if(board_direction==1) i=6-ix;
                if(board_direction!=1) i=7+ix;
            }
        } else if(is_in_right) {
            if(is_in_up) {
                if(board_direction==1) i=18+ix;
                if(board_direction!=1) i=19-ix;
            } else if(is_in_down) {
                if(board_direction==1) i=7-ix;
                if(board_direction!=1) i=6+ix;
            }
        }
        return i;
    }

    if(!pressed){
        if(is_on_point) {
            i=find_point_index(ix, iy); 
            if(abs(iy)==1 && abs(pos_ptr->checker[i])>=5) {
                pos_ptr->checker[i] += player;
            } else { pos_ptr->checker[i] = fill_point(iy); }
        } else if(is_on_bar) {
            if(is_in_up) {
                if(!is_in_uplabel) {
                    if(abs(iy)==5 && abs(pos_ptr->checker[25])>=5) {
                        pos_ptr->checker[25] += 1;
                    } else { pos_ptr->checker[25] = iy; }
                } else {pos_ptr->checker[25] = 0; }
            } else if(is_in_down) {
                if(!is_in_downlabel) {
                    if(abs(iy)==5 && abs(pos_ptr->checker[0])>=5) {
                        pos_ptr->checker[0] -= 1;
                    } else { pos_ptr->checker[0] = iy; }
                } else { pos_ptr->checker[0] = 0; }
            } else if(is_in_center) {
                pos_ptr->checker[25] = 0;
                pos_ptr->checker[0] = 0;
            } else { printf("ERROR! Cas impossible!\n"); }
        }
    }

    if(!pressed){
        if(is_in_player1_roll){
            pos_ptr->cube_action=1;
            pos_ptr->player_on_roll=PLAYER1;
        } else if(is_in_player2_roll){
            pos_ptr->cube_action=1;
            pos_ptr->player_on_roll=PLAYER2;
        } else if(is_in_player1_dice1){
            pos_ptr->cube_action=0;
            pos_ptr->player_on_roll=PLAYER1;
            if(button==IUP_BUTTON1)pos_ptr->dice[0]+=1;
            if(button==IUP_BUTTON3)pos_ptr->dice[0]-=1;
            if(pos_ptr->dice[0]>6) pos_ptr->dice[0]=6;
            if(pos_ptr->dice[0]<1) pos_ptr->dice[0]=1;
        } else if(is_in_player1_dice2){
            pos_ptr->cube_action=0;
            pos_ptr->player_on_roll=PLAYER1;
            if(button==IUP_BUTTON1)pos_ptr->dice[1]+=1;
            if(button==IUP_BUTTON3)pos_ptr->dice[1]-=1;
            if(pos_ptr->dice[1]>6) pos_ptr->dice[1]=6;
            if(pos_ptr->dice[1]<1) pos_ptr->dice[1]=1;
        } else if(is_in_player2_dice1){
            pos_ptr->cube_action=0;
            pos_ptr->player_on_roll=PLAYER2;
            if(button==IUP_BUTTON1)pos_ptr->dice[0]+=1;
            if(button==IUP_BUTTON3)pos_ptr->dice[0]-=1;
            if(pos_ptr->dice[0]>6) pos_ptr->dice[0]=6;
            if(pos_ptr->dice[0]<1) pos_ptr->dice[0]=1;
        } else if(is_in_player2_dice2){
            pos_ptr->cube_action=0;
            pos_ptr->player_on_roll=PLAYER2;
            if(button==IUP_BUTTON1)pos_ptr->dice[1]+=1;
            if(button==IUP_BUTTON3)pos_ptr->dice[1]-=1;
            if(pos_ptr->dice[1]>6) pos_ptr->dice[1]=6;
            if(pos_ptr->dice[1]<1) pos_ptr->dice[1]=1;
        }
    }
    printf("dice: %i %i\n", pos_ptr->dice[0], pos_ptr->dice[1]);

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
                && !is_in_cube_positions && !is_in_dice_positions) {
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
                update_sb_msg(msg_err_failed_to_create_db);
                printf("Database creation failed\n");
                return result;
            }
            /* int result = db_open(db_filename); */
            update_sb_msg(msg_info_db_created);
            draw_canvas(cdv);
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
                update_sb_msg(msg_err_failed_to_open_db);
                printf("%s\n",msg_err_failed_to_open_db);
                return result;
            }
            db_select_position(db, &pos_nb,
                    pos_list_id, pos_list);
            db_select_libraries(db, &lib_nb, lib_list_id, lib_list);
            goto_first_position_cb();
            update_sb_lib();
            update_sb_msg(msg_info_db_loaded);
            printf("%s\n",msg_info_db_loaded);
            break; 

        case -1 : 
            printf("IupFileDlg: Operation Canceled");
            return 1;
            break; 
    }

    IupDestroy(filedlg);

    draw_canvas(cdv);

    return IUP_DEFAULT;

}

static int item_import_action_cb(void)
{
    printf("\nitem_import_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    Ihandle *filedlg;
    filedlg=IupFileDlg();
    IupSetAttribute(filedlg, "DIALOGTYPE", "OPEN");
    IupSetAttribute(filedlg, "TITLE", "Import Position or Match");
    IupSetAttribute(filedlg, "EXTFILTER",
            "Position File (.txt)|*.txt|");
    IupPopup(filedlg, IUP_CENTER, IUP_CENTER);

    switch(IupGetInt(filedlg, "STATUS"))
    {
        case 1: //new file
            printf("Position or Match does not exist.");
            break;
        case 0: //file already exists
            const char *p_filename=IupGetAttribute(filedlg,"VALUE");
            FILE *f=open_input(p_filename);
            if(f==NULL){
                update_sb_msg(msg_err_failed_to_import_pos);
                printf("%s\n",msg_err_failed_to_import_pos);
            }
            parse_from_file=1;
            txtfiletype_t rc_txtft=check_if_file_is_position_or_match(f);
            int rc;
            switch(rc_txtft){
                case TXT_POSITION:
                    int pid;
                    rc=db_import_position_from_file(db,f,&pid);
                    if(rc){
                        db_select_position(db,&pos_nb,pos_list_id,pos_list);
                        goto_position_cb(&pid);
                        switch_to_library("main",&lib_index);
                        update_sb_msg(msg_info_position_imported);
                    } else{
                        update_sb_msg(msg_err_failed_to_import_pos);
                    }
                    break;
                case TXT_MATCH:
                    rc=db_import_match_from_file(db,f);
                    if(rc){
                        db_select_position(db,&pos_nb,pos_list_id,pos_list);
                        goto_first_position_cb();
                        switch_to_library("main",&lib_index);
                        update_sb_msg(msg_info_match_imported);
                    } else{
                        update_sb_msg(msg_err_failed_to_import_match);
                    }
                    break;
                default:
                    update_sb_msg(msg_err_not_pos_nor_match_file);
                    printf(msg_err_not_pos_nor_match_file);
                    break;
            }
            break;
        case -1:
            printf("IupFileDlg: Operation Canceled");
            return 1;
            break;
    }

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
    if(db!=NULL) db_close(db);
    IupExitLoop();
    return EXIT_SUCCESS;
}

static int item_copy_action_cb(void)
{
    printf("\nitem_copy_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    Ihandle* clipboard=IupClipboard();
    Ihandle* text=IupText(NULL);
    char _xgid[1000];_xgid[0]='\0';
    convert_position_to_xgid(pos_ptr,_xgid);
    IupSetAttribute(text,"VALUE",_xgid);
    IupSetAttribute(clipboard, "TEXT", IupGetAttribute(text,"VALUE"));
    IupDestroy(clipboard);
    return IUP_DEFAULT;
}

static int item_paste_action_cb(void)
{
    printf("\nitem_paste_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    parse_from_file=0;
    Ihandle* clipboard=IupClipboard();
    Ihandle* text=IupText(NULL);
    IupSetAttribute(text, "VALUE", IupGetAttribute(clipboard, "TEXT"));
    IupDestroy(clipboard);
    char* ctext=IupGetAttribute(text,"VALUE");
    printf("clipboard\n%s\n",ctext);

    char token[LINE_NBMAX][LINE_LENGTHMAX]; int i;
    char *c = strtok(ctext, "\n");
    token[0][0]='\0'; i=0; while(c!=NULL){
        strcat(token[i],c); i+=1;
        c=strtok(NULL, "\n");
        token[i][0]='\0';
    }
    for(int j=0;j<i;j++){
        printf("tok %i: %s\n",j,token[j]);
    }
    int pid;
    int rc=db_import_position_from_lines(db,token,i,&pid);
    if(rc){
    db_select_position(db,&pos_nb,pos_list_id,pos_list);
    goto_position_cb(&pid);
    switch_to_library("main",&lib_index);
    update_sb_msg(msg_info_position_imported);
    } else{
        update_sb_msg(msg_err_failed_to_import_pos);
    }

    return IUP_DEFAULT;
}

static int item_editmode_action_cb(void)
{
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    toggle_editmode_cb();
    return IUP_DEFAULT;
}

static int item_nextposition_action_cb(void)
{
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return 0;
    }
    goto_next_position_cb();
    return IUP_DEFAULT;
}

static int item_prevposition_action_cb(void)
{
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return 0;
    }
    goto_prev_position_cb();
    return IUP_DEFAULT;
}

static int item_newposition_action_cb(void)
{
    printf("\nitem_newlibrary_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    bool exist=false;
    int nb=0;
    int _id[1000];
    db_find_identical_position(db, pos_ptr, &exist, &nb, _id);
    if(exist){
        goto_position_cb(&_id[0]);
        update_sb_msg(msg_info_position_already_exists);
        printf("Position already exists. nb: %i\n", nb);
        for(int i=0;i<nb;i++) printf("_id: %i\n", _id[i]);
    } else {
        db_insert_position(db, pos_ptr);
        update_sb_msg(msg_info_position_written);
        db_select_position(db, &pos_nb,
                pos_list_id, pos_list);
        goto_last_position_cb();
    }
    return IUP_DEFAULT;
}

static int item_updateposition_action_cb(void)
{
    printf("\nitem_updateposition_action_cb\n");
    bool exist=false;
    int nb=0;
    int _id[1000];
    db_find_identical_position(db, pos_ptr, &exist, &nb, _id);
    if(exist){
        goto_position_cb(&_id[0]);
        update_sb_msg(msg_info_position_already_exists);
        printf("Position already exists. nb: %i\n", nb);
        for(int i=0;i<nb;i++) printf("_id[%i]: %i\n",i, _id[i]);
    } else {
        int id=pos_list_id[pos_index];
        if(db_analysis_exist(db,id)){
            goto_position_cb(&id);
            mode_active=NORMAL;
            update_sb_msg(msg_err_cannot_update_position_with_analysis);
            update_sb_mode();
        } else {
            db_update_position(db, &id, pos_ptr);
            db_select_position(db, &pos_nb,
                    pos_list_id, pos_list);
            mode_active=NORMAL;
            update_sb_msg(msg_info_position_updated);
            update_sb_mode();
        }
    }
    return IUP_DEFAULT;
}

static int item_importposition_action_cb(void)
{
    printf("\nitem_importposition_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    Ihandle *filedlg;
    filedlg=IupFileDlg();
    IupSetAttribute(filedlg, "DIALOGTYPE", "OPEN");
    IupSetAttribute(filedlg, "TITLE", "Import Position");
    IupSetAttribute(filedlg, "EXTFILTER",
            "Position File (.txt)|*.txt|");
    IupPopup(filedlg, IUP_CENTER, IUP_CENTER);

    switch(IupGetInt(filedlg, "STATUS"))
    {
        case 1: //new file
            printf("Position does not exist.");
            break;
        case 0: //file already exists
            const char *p_filename=IupGetAttribute(filedlg,"VALUE");
            FILE *f=open_input(p_filename);
            if(f==NULL){
                update_sb_msg(msg_err_failed_to_import_pos);
                printf("%s\n",msg_err_failed_to_import_pos);
            }
            parse_from_file=1;
            int pid;
            int rc=db_import_position_from_file(db,f,&pid);
            if(rc){
                db_select_position(db,&pos_nb,pos_list_id,pos_list);
                goto_position_cb(&pid);
                switch_to_library("main",&lib_index);
                update_sb_msg(msg_info_position_imported);
            } else{
                update_sb_msg(msg_err_failed_to_import_pos);
            }
            break;
        case -1:
            printf("IupFileDlg: Operation Canceled");
            return 1;
            break;
    }
    return IUP_DEFAULT;
}

static int item_newlibrary_action_cb(void)
{
    error_callback();
    return IUP_DEFAULT;
}

static int item_deletelibrary_action_cb(void)
{
    printf("\nitem_deletelibrary_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    int ilib=0; char lname[LIBRARY_NAME_MAX];lname[0]='\0';
    char s[10000];s[0]='\0';
    char t[LIBRARY_NAME_MAX];t[0]='\0';
    strcat(s,"name %l|");
    for(int i=0;i<lib_nb;i++){
        sprintf(t,"%s|",lib_list[i]);
        strcat(s,t);
    }
    strcat(s,"\n");
    if (!IupGetParam("Delete Library", NULL, 0,
                s,&ilib)){}
    printf("ilib %i %s\n", ilib, lib_list[ilib]);
    strcat(lname,lib_list[ilib]);
    db_delete_library(db,lname);
    sprintf(t, "%s has been deleted.",lname);
    update_sb_msg(t);
    return IUP_DEFAULT;
}

static int item_addtolibrary_action_cb(void)
{
    printf("\nitem_addtolibrary_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    char lname[100]; lname[0]='\0';
    if (!IupGetParam("Add Position to Library", NULL, 0,
                "name %s\n",
                &lname)){}
    printf("library name: %s\n",lname);
    int pos_id = pos_list_id[pos_index];
    add_position_to_library(db,pos_id,lname); 
    return IUP_DEFAULT;
}

static int item_gotolibrary_action_cb(void)
{
    printf("\nitem_gotolibrary_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    int ilib=0; char lname[LIBRARY_NAME_MAX];lname[0]='\0';
    char s[10000];s[0]='\0';
    char t[LIBRARY_NAME_MAX];t[0]='\0';
    strcat(s,"name %l|");
    for(int i=0;i<lib_nb;i++){
        sprintf(t,"%s|",lib_list[i]);
        strcat(s,t);
    }
    strcat(s,"\n");
    if (!IupGetParam("Go To Library", NULL, 0,
                s,&ilib)){}
    printf("ilib %i %s\n", ilib, lib_list[ilib]);
    strcat(lname,lib_list[ilib]);
    db_select_position_from_specific_library(db, lname,
            &pos_nb, pos_list_id, pos_list);
    int l_id;
    if(db_library_exists(db,lname)){
        db_get_library_id_from_name(db,lname,&l_id);
        lib_index=find_index_from_int(l_id, lib_list_id, lib_nb);
        update_sb_lib();
        char t[100]; t[0]='\0'; sprintf(t, "Switched to %s.",lib_list[lib_index]);
        update_sb_msg(t);
    } else {
        db_select_position(db, &pos_nb, pos_list_id, pos_list);
        lib_index=LIBRARIES_NUMBER_MAX-1; //main lib
        update_sb_lib();
        char t[100]; t[0]='\0'; sprintf(t, "Switched to %s.",lib_list[lib_index]);
    }

    if(pos_nb==0){ //if no position found in lib
        pos_list[0]=POS_DEFAULT;
        pos_list_id[0]=-1;
        pos_nb=1;
    }
    goto_first_position_cb();
    return IUP_DEFAULT;
}

static int item_importmatch_action_cb(void)
{
    printf("\nitem_importmatch_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    Ihandle *filedlg;
    filedlg=IupFileDlg();
    IupSetAttribute(filedlg, "DIALOGTYPE", "OPEN");
    IupSetAttribute(filedlg, "TITLE", "Import Match");
    IupSetAttribute(filedlg, "EXTFILTER",
            "Position File (.txt)|*.txt|");
    IupPopup(filedlg, IUP_CENTER, IUP_CENTER);

    switch(IupGetInt(filedlg, "STATUS"))
    {
        case 1: //new file
            printf("Match does not exist.");
            break;
        case 0: //file already exists
            const char *p_filename=IupGetAttribute(filedlg,"VALUE");
            FILE *f=open_input(p_filename);
            if(f==NULL){
                update_sb_msg(msg_err_failed_to_import_match);
                printf("%s\n",msg_err_failed_to_import_match);
            }
            parse_from_file=1;
            int pid;
            int rc=db_import_position_from_file(db,f,&pid);
            if(rc){
                db_select_position(db,&pos_nb,pos_list_id,pos_list);
                goto_position_cb(&pid);
                switch_to_library("main",&lib_index);
                update_sb_msg(msg_info_position_imported);
            } else{
                update_sb_msg(msg_err_failed_to_import_match);
            }
            break;
        case -1:
            printf("IupFileDlg: Operation Canceled");
            return 1;
            break;
    }
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

static int item_searchengine_action_cb(void)
{

    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }

    int force_cube=0;
    int cube_owner=1;
    int cube_value=1;
    int force_score=0;
    int score_type=1;
    int score_player1=7;
    int score_player2=7;
    int criteria_abspipcount=0;
    int Pmin=-30, Pmax=30;
    int criteria_checkeroff=0;
    int omin=0;
    int criteria_backchecker1=0; int bc_num1=0;
    int criteria_backchecker2=0; int bc_num2=0;
    int criteria_zone1=0; int z_num1=0;
    int criteria_zone2=0; int z_num2=0;
    int criteria_blunder=0; int bmin=0; int bmax=1000;
    int criteria_equity=0; int emin=-500; int emax=500;
    if (!IupGetParam("Search Position", NULL, 0,
                "Bt %u[, MyCancel ]\n"
                "CUBE%b[,]\n"
                "player%o|Center|Player 1|Player 2|\n"
                "value%l|1|2|4|8|16|32|\n"
                "%t\n"
                "SCORE%b[,]\n"
                "type %o|match|unlimited|\n"
                "player 1 (away, 0=post-Crawford, 1=Crawford) %i[0,25]\n"
                "player 2 (away, 0=post-Crawford, 1=Crawford) %i[0,25]\n"
                "%t\n"
                "PIPCOUNT%b[,]\n"
                "min %i\n"
                "max %i\n"
                "%t\n"
                "CHECKER OFF %b[,]\n"
                "min %i[0,15]\n"
                "%t\n"
                "BACKCHECKER PLAYER 1 %b[,]\n"
                "%i[0,15]\n"
                "BACKCHECKER PLAYER 2 %b[,]\n"
                "%i[0,15]\n"
                "%t\n"
                "CHECKERZONE PLAYER 1 %b[,]\n"
                "%i[0,15]\n"
                "CHECKERZONE PLAYER 2 %b[,]\n"
                "%i[0,15]\n"
                "%t\n"
                "BLUNDER %b[,]\n"
                "min%i\n"
                "max%i\n"
                "%t\n"
                "EQUITY %b[,]\n"
                "min%i\n"
                "max%i\n"
                "",
                &force_cube,&cube_owner,&cube_value,
                &force_score,&score_type,
                &score_player1,&score_player2,
                &criteria_abspipcount,&Pmin,&Pmax,
                &criteria_checkeroff,&omin,
                &criteria_backchecker1,&bc_num1,
                &criteria_backchecker2,&bc_num2,
                &criteria_zone1,&z_num1,
                &criteria_zone2,&z_num2,
                &criteria_blunder,&bmin,&bmax,
                &criteria_equity,&emin,&emax,
                NULL)){}
                
    printf("cube: active %i owner %i value %i\n",
            force_cube,cube_owner,cube_value);
    printf("score: active %i type %i p1_score %i p2_score %i\n",
            force_score,score_type,score_player1,score_player2);
    printf("pipcount: active %i min %i max %i\n",
            criteria_abspipcount,Pmin,Pmax);
    printf("backchecker1: active %i n %i\n",
            criteria_backchecker1, bc_num1);
    printf("backchecker2: active %i n %i\n",
            criteria_backchecker2, bc_num2);
    printf("checkeroff: active %i min %i\n",
            criteria_checkeroff,omin);
    printf("checkerzone1: active %i n %i\n",
            criteria_zone1,z_num1);
    printf("checkerzone2: active %i n %i\n",
            criteria_zone2,z_num2);
    printf("blunder: active %i min %i max %i\n",
            criteria_blunder,bmin,bmax);
    printf("equity: active %i min %i max %i\n",
            criteria_equity,emin,emax);

    copy_position(&POS_VOID,&pos_buffer);
    pos_ptr=&pos_buffer;
    if(force_cube){
        switch(cube_owner){
            case 0:
                pos_ptr->cube=0;
                break;
            case 1:
                pos_ptr->cube=cube_value;
                break;
            case 2:
                pos_ptr->cube=-cube_value;
                break;
        }
        printf("cube: %i\n", pos_ptr->cube);
    }
    if(force_score){
        switch(score_type){
            case 0:
                pos_ptr->p1_score=score_player1;
                pos_ptr->p2_score=score_player2;
                break;
            case 1:
                pos_ptr->p1_score=-1;
                pos_ptr->p2_score=-1;
                break;
        }
    }
    db_select_specific_position(db, pos_ptr,
            force_cube, force_score,
            criteria_blunder, bmin, bmax,
            criteria_equity, emin, emax,
            &pos_nb, pos_list_id, pos_list);
    if(criteria_checkeroff) filter_position_by_checkeroff(omin);
    if(criteria_abspipcount) filter_position_by_pipcount(Pmin,Pmax,true);
    if(criteria_backchecker1) filter_position_by_backchecker(PLAYER1,bc_num1);
    if(criteria_backchecker2) filter_position_by_backchecker(PLAYER2,bc_num2);
    if(criteria_zone1) filter_position_by_checker_in_the_zone(PLAYER1,z_num1);
    if(criteria_zone2) filter_position_by_checker_in_the_zone(PLAYER2,z_num2);
    if(pos_nb==0){
        pos_list[0]=POS_DEFAULT;
        pos_list_id[0]=-1;
        pos_nb=1;
    }
    goto_first_position_cb();


    return IUP_DEFAULT;
}

static int item_findpositionwithoutanalysis_action_cb(void)
{
    printf("\nitem_findpositionwithoutanalysis_action_cb\n");
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    int j=0;
    for(int i=0;i<pos_nb;i++){
        if(!db_analysis_exist(db,pos_list_id[i])){
            pos_list_tmp[j]=pos_list[i];
            pos_list_id_tmp[j]=pos_list_id[i];
            j+=1;
        }
    }
    for(int i=0;i<j;i++){
            pos_list[i]=pos_list_tmp[i];
            pos_list_id[i]=pos_list_id_tmp[i];
    }
    pos_nb=j;
    lib_index=LIBRARIES_NUMBER_MAX-2; //mix lib
    update_sb_lib();
    char t[100]; t[0]='\0';
    sprintf(t, "Found %d position(s) without analysis.",pos_nb);
    update_sb_msg(t);
    goto_first_position_cb();
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

static int toggle_editmode_cb()
{
    if(db==NULL){
        update_sb_msg(msg_info_no_db_loaded);
        return IUP_DEFAULT;
    }

    if(mode_active != EDIT) {
        mode_active=EDIT;
        is_pointletter_active=true;
        copy_position(pos_ptr, &pos_buffer);
        pos_ptr=&pos_buffer;
        draw_canvas(cdv);
        set_visibility_off(cmdline);
    } else {
        mode_active=NORMAL;
        is_pointletter_active=false;
        draw_canvas(cdv);
    }
    IupSetAttribute(sb_mode, "TITLE", mode_to_str(mode_active));
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int toggle_cmdmode_cb()
{
    if(mode_active != CMD){
        mode_active=CMD;
        set_visibility_on(cmdline);
        IupSetAttribute(cmdline, "INSERT", ":");
        draw_canvas(cdv);
        IupSetFocus(cmdline);
    } else {
        mode_active=NORMAL;
        set_visibility_off(cmdline);
        is_pointletter_active=false;
        /* for(int i=0;i<100;i++) cmdtext[i]='\0'; */
        cmdtext = IupGetAttribute(cmdline, "VALUE");
        IupSetAttribute(cmdline, "VALUE", "");
        draw_canvas(cdv);
        parse_cmdline(cmdtext);
    }
    IupSetAttribute(sb_mode, "TITLE", mode_to_str(mode_active));
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int toggle_analysis_visibility_cb()
{
    if(db==NULL){
        update_sb_msg(msg_err_no_db_opened);
        return IUP_DEFAULT;
    }
    toggle_visibility_cb(analysis);
    char* att = IupGetAttribute(analysis, "VISIBLE");
    if(strcmp(att,"NO") == 0) {
        IupSetAttribute(split, "VALUE", "1000");
    } else if (strcmp(att,"YES") == 0) {
        IupSetAttribute(split, "VALUE", DEFAULT_SPLIT_VALUE);
    }
    IupRefresh(dlg);
    return IUP_DEFAULT;
}

static int set_visibility_on(Ihandle* ih){
    IupSetAttribute(ih, "VISIBLE", "YES");
    IupSetAttribute(ih, "FLOATING", "NO");
    IupRefresh(ih);
    return IUP_DEFAULT;
}

static int set_visibility_off(Ihandle* ih){
    IupSetAttribute(ih, "VISIBLE", "NO");
    IupSetAttribute(ih, "FLOATING", "YES");
    IupRefresh(ih);
    return IUP_DEFAULT;
}

static int toggle_visibility_cb(Ihandle* ih)
{
    char* att = IupGetAttribute(ih, "VISIBLE");
    if(strcmp(att,"NO")==0) {
        printf("display panel\n");
        IupSetAttribute(ih, "VISIBLE", "YES");
        IupSetAttribute(ih, "FLOATING", "NO");
    } else if(strcmp(att, "YES")==0) {
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

static int minus_cb(Ihandle* ih, int c){
    printf("\nminus_cb\n");
    printf("key_m %c\n", key_m);
    switch (mode_active) {
        case EDIT:
            if(key_m==-1){
                is_score_to_fill=true;
                digit_m[0]='-';
                key_m=c;
            } else if(isalpha(key_m)){
                is_point_to_fill=true;
                key_m=c;
            }
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int bracketleft_cb(Ihandle* ih, const int c){
    printf("\nbracketleft_cb\n");
    printf("key_m %c\n", key_m);
    switch (mode_active) {
        case EDIT:
            if(isdigit(key_m)){
                digit_m[2]='\0';
                int i = atoi(digit_m);
                printf("p1_score i: %i\n", i);
                pos_ptr->p1_score = i;
                draw_canvas(cdv);
            }
            key_m=-1;
            is_score_to_fill=false;
            for(int i=0; i<4; i++) digit_m[i]='\0';
            printf("digit_m %s\n", digit_m);
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int bracketright_cb(Ihandle* ih, int c){
    printf("\nbracketright_cb\n");
    switch (mode_active) {
        case EDIT:
            printf("key_m %c\n", key_m);
            if(isdigit(key_m)){
                digit_m[2]='\0';
                int i = atoi(digit_m);
                printf("p2_score i: %i\n", i);
                pos_ptr->p2_score = i;
                draw_canvas(cdv);
            }
            key_m=-1;
            is_score_to_fill=false;
            for(int i=0; i<4; i++) digit_m[i]='\0';
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int cr_cb(Ihandle* ih, int c){
    printf("\ncr_cb\n");
    switch(mode_active) {
        case EDIT:
            make_point=!make_point;
            key_m=-1;
            break;
        case CMD:
            toggle_cmdmode_cb();
            break;
        case NORMAL:
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int esc_cb(Ihandle* ih, int c){
    printf("\nesc_cb\n");
    switch(mode_active) {
        case EDIT:
            make_point=false;
            is_score_to_fill=false;
            is_point_to_fill=false;
            is_cube_to_fill=false;
            key_m=-1;
            sign_m=1;
            for(int i=0;i<4;i++) digit_m[i]='\0';
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int backspace_cb(Ihandle* ih, int c){
    switch(mode_active) {
        case(EDIT):
            for(int i=0; i<26; i++) pos_ptr->checker[i]=0;
            draw_canvas(cdv);
            key_m=-1;
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int space_cb(Ihandle* ih, int c){
    switch(mode_active) {
        case(NORMAL):
        case(EDIT):
            toggle_cmdmode_cb(cmdline);
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

int refresh_position(){
    draw_canvas(cdv);
    update_sb_lib();
    if(pos_list[pos_index].cube_action==0){
        update_checker_analysis(pos_list_id[pos_index]);
    } else {
        update_cube_analysis(pos_list_id[pos_index]);
    }
    return 1;
}

static int goto_first_position_cb(){
    get_first_position();
    refresh_position();
    return 1;
}

static int goto_prev_position_cb(){
    get_prev_position();
    refresh_position();
    return 1;
}

static int goto_next_position_cb(){
    get_next_position();
    refresh_position();
    return 1;
}

static int goto_last_position_cb(){
    get_last_position();
    refresh_position();
    return 1;
}

static int goto_position_cb(int* id){
    get_position(id);
    refresh_position();
    return 1;
}
static int left_cb(Ihandle* ih, int c){
    printf("\nleft_cb\n");
    switch(mode_active) {
        case(NORMAL):
            goto_prev_position_cb();
            break;
        case(EDIT):
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int right_cb(Ihandle* ih, int c){
    printf("\nright_cb\n");
    switch(mode_active) {
        case(NORMAL):
            goto_next_position_cb();
            break;
        case(EDIT):
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int pgup_cb(Ihandle* ih, int c){
    printf("\npgup_cb\n");
    switch(mode_active) {
        case(NORMAL):
            goto_first_position_cb();
            break;
        case(EDIT):
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int pgdn_cb(Ihandle* ih, int c){
    printf("\npgdn_cb\n");
    switch(mode_active) {
        case(NORMAL):
            goto_last_position_cb();
            break;
        case(EDIT):
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int j_cb(Ihandle* ih, int c){
    switch(mode_active) {
        case(NORMAL):
            goto_next_position_cb();
            break;
        case(EDIT):
            editmode_letter_cb(ih, c);
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int k_cb(Ihandle* ih, int c){
    switch(mode_active) {
        case(NORMAL):
            goto_prev_position_cb();
            break;
        case(EDIT):
            editmode_letter_cb(ih, c);
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int G_cb(Ihandle* ih, int c){
    switch(mode_active) {
        case(NORMAL):
            goto_last_position_cb();
            break;
        case(EDIT):
            editmode_letter_cb(ih, c);
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int editmode_letter_cb(Ihandle* ih, int c){
    printf("editmode_letter_cb %c\n", c);
    if(mode_active!=EDIT) return 0;

    void f(const char c, int* i, int* sign){
        if(tolower(c)!='z') { //point
            is_point_to_fill=true;
            is_cube_to_fill=false;
            key_m=c;
            if(islower(c)) {
                *i=char_in_string(c,PLAYER1_POINTLABEL);
                *sign=1;
            } else {
                *i=char_in_string(c,PLAYER2_POINTLABEL);
                *sign=-1;
            }
        } else { //cube
            is_point_to_fill=false;
            is_cube_to_fill=true;
            if(islower(c)) *sign=1;
            if(isupper(c)) *sign=-1;
            key_m=c;
        }
    }

    printf("key_m %c\n", key_m);
    if(key_m==-1) {
        f(c, &point_m, &sign_m);
        printf("point_m %i\n", point_m);
        printf("sign_m %i\n", sign_m);
    } else if(key_m=='-'){
        f(c, &point_m2, &sign_m);
        int i1=fmin(point_m, point_m2);
        int i2=fmax(point_m, point_m2);
        if(make_point) {
            for(int k=i1; k<=i2; k++) {
                pos_ptr->checker[k]=sign_m*2;
            }
        } else {
            for(int k=i1; k<=i2; k++) {
                pos_ptr->checker[k]+=sign_m;
            }
        }
        draw_canvas(cdv);
        is_point_to_fill=false;
        key_m=-1;
    }

    return IUP_DEFAULT;
}

static int editmode_digit_cb(Ihandle* ih, int c){
    printf("\neditmode_digit_cb %c\n", c);
    int i; int n; char s[2]; s[0]=c; s[1]='\0';
    n = atoi(s);
    switch (mode_active) {
        case EDIT:
            printf("key_m %c\n", key_m);
            if(key_m==-1) {
                printf("-1\n");
                is_score_to_fill=true;
                digit_m[0]=c;
                key_m=c;
            } else if(isdigit(key_m)) {
                printf("digit\n");
                digit_m[1]=c;
                key_m=c;
            } else if(key_m=='-') {
                printf("minus\n");
                if(is_point_to_fill) {
                    pos_ptr->checker[point_m]-=sign_m*n;
                    is_point_to_fill=false;
                    key_m=-1;
                    draw_canvas(cdv);
                } else if(is_score_to_fill) {
                    digit_m[0]='-';
                    digit_m[1]=c;
                    key_m=c;
                }
            } else if(isalpha(key_m)) {
                printf("alpha\n");
                if(tolower(key_m)!='z'){
                pos_ptr->checker[point_m]+=sign_m*n;
                if(n==0) pos_ptr->checker[point_m]=0;
                draw_canvas(cdv);
                is_point_to_fill=false;
                } else {
                    pos_ptr->cube=sign_m*n;
                    is_cube_to_fill=false;
                    draw_canvas(cdv);
                }
                key_m=-1;
            }
            break;
        default:
            break;
    }
    return IUP_DEFAULT;
}

static int board_direction_left_cb(void){
    board_direction = -BOARD_DIRECTION;
    draw_canvas(cdv);
    return IUP_DEFAULT;
}

static int board_direction_right_cb(void){
    board_direction = BOARD_DIRECTION;
    draw_canvas(cdv);
    return IUP_DEFAULT;
}

static int display_player_on_roll_down(){
    printf("\ndisplay_player_on_roll_down\n");
    if(pos_ptr->player_on_roll==PLAYER2){
        invert_position(pos_ptr);
    }
    refresh_position();
}

static int display_player_on_roll_up(){
    printf("\ndisplay_player_on_roll_up\n");
    if(pos_ptr->player_on_roll==PLAYER1){
        invert_position(pos_ptr);
    }
    refresh_position();
}

// END Callbacks



/************************ Main ****************************/
int main(int argc, char **argv)
{
    // initialization
    pos = POS_DEFAULT;
    pos_ptr = &pos;
    pos_prev_ptr = &pos;
    pos_next_ptr = &pos;
    pos_nb = 0;
    pos_index = 0;
    lib_list[LIBRARIES_NUMBER_MAX-2][0]='\0';
    strcat(lib_list[LIBRARIES_NUMBER_MAX-2], "mix");
    lib_list[LIBRARIES_NUMBER_MAX-1][0]='\0';
    strcat(lib_list[LIBRARIES_NUMBER_MAX-1], "main");
    lib_index=LIBRARIES_NUMBER_MAX-1;
    lib_nb=1;
    ca[0]=CA_VOID;
    ca_index=0;
    ca_ptr = &ca[0];
    da=D_VOID;
    da_ptr=&da;


    IupOpen(&argc, &argv);
    IupControlsOpen();
    IupImageLibOpen();
    IupSetLanguage("ENGLISH");

    menu = create_menus();
    toolbar = create_toolbar();

    canvas = create_canvas();
    analysis = create_analysis();
    position = IupHbox(canvas, NULL);

    statusbar = create_statusbar();
    cmdline = create_cmdline();

    split = IupSplit(position, analysis);
    IupSetAttribute(split, "ORIENTATION", "HORIZONTAL");
    IupSetAttribute(split, "VALUE", "1000");
    /* IupSetAttribute(split, "MINMAX", DEFAULT_SPLIT_MINMAX); */
    /* IupSetAttribute(split, "AUTOHIDE", "YES"); */

    vbox = IupVbox(toolbar, split, statusbar, cmdline, NULL);
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

    if(db!=NULL) db_close(db);
    IupClose();

    return EXIT_SUCCESS;
}
