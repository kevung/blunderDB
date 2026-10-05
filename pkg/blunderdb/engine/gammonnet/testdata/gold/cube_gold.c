/* Produces the gold file for blunderDB's Go port of gammonNet's cube model
 * (gn_cube.c / gn_met.c). Reads testdata/cube_corpus.bin, writes
 * testdata/cube_gold.bin. See README.md for the build recipe.
 *
 * Each case is decided twice: gn_cube_decide (no beaver) and
 * gn_cube_decide_ex with the beaver flag set. The plain fields of the two
 * must be identical bit for bit -- the harness refuses to write otherwise --
 * and the record carries the plain fields followed by the beaver answer.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "gn_infer.h"
#include "gn_cube.h"
#include "gn_met.h"

/* Record layout, little-endian, fixed 56 bytes -- see cube_gold_test.go's
 * writer for the authoritative field order. */
typedef struct {
    float probs[GN_NUM_OUTPUTS]; /* 20 */
    int32_t owner;                /* 4  -- GnCubeOwner */
    double efficiency;            /* 8  */
    int32_t jacoby;                /* 4  */
    int32_t has_state;             /* 4  -- 0 = money, 1 = match */
    int32_t away_on_roll;          /* 4  */
    int32_t away_opponent;         /* 4  */
    int32_t cube;                   /* 4  */
    int32_t crawford;               /* 4  */
} Case; /* 56 bytes */

int main(int argc, char **argv) {
    if (argc < 3) { fprintf(stderr, "usage: cube_gold corpus.bin out.bin\n"); return 2; }
    FILE *f = fopen(argv[1], "rb");
    if (!f) { perror("corpus"); return 1; }
    char magic[4]; int32_t count;
    if (fread(magic,1,4,f)!=4 || memcmp(magic,"GNCB",4) || fread(&count,4,1,f)!=1) {
        fprintf(stderr,"bad corpus\n"); return 1; }

    FILE *o = fopen(argv[2], "wb");
    if (!o) { perror("out"); return 1; }
    fwrite("GNCH",1,4,o); fwrite(&count,4,1,o);

    for (int32_t i = 0; i < count; i++) {
        Case c;
        if (fread(&c.probs,4,GN_NUM_OUTPUTS,f)!=(size_t)GN_NUM_OUTPUTS) { fprintf(stderr,"short corpus at %d\n",i); return 1; }
        if (fread(&c.owner,4,1,f)!=1 || fread(&c.efficiency,8,1,f)!=1 ||
            fread(&c.jacoby,4,1,f)!=1 || fread(&c.has_state,4,1,f)!=1 ||
            fread(&c.away_on_roll,4,1,f)!=1 || fread(&c.away_opponent,4,1,f)!=1 ||
            fread(&c.cube,4,1,f)!=1 || fread(&c.crawford,4,1,f)!=1) {
            fprintf(stderr,"short corpus at %d\n",i); return 1; }

        GnMatchState state;
        GnMatchState *sp = NULL;
        if (c.has_state) {
            state.away_on_roll = c.away_on_roll;
            state.away_opponent = c.away_opponent;
            state.cube = c.cube;
            state.crawford = c.crawford;
            sp = &state;
        }

        GnCubeDecision dec, bv;
        int rc = gn_cube_decide(c.probs, (GnCubeOwner)c.owner, sp, c.efficiency, c.jacoby, &dec);
        int rcb = gn_cube_decide_ex(c.probs, (GnCubeOwner)c.owner, sp, c.efficiency, c.jacoby, 1, &bv);
        if (rc != rcb || (rc == 0 && (dec.action != bv.action ||
            memcmp(&dec.equity_no_double, &bv.equity_no_double, 8) ||
            memcmp(&dec.equity_double, &bv.equity_double, 8) ||
            memcmp(&dec.take_point, &bv.take_point, 8)))) {
            fprintf(stderr, "case %d: the beaver flag moved a plain field\n", i); return 1; }

        int32_t ok = (rc == 0) ? 1 : 0;
        int32_t action = ok ? (int32_t)dec.action : 0;
        double eND = ok ? dec.equity_no_double : 0.0;
        double eD = ok ? dec.equity_double : 0.0;
        double tp = ok ? dec.take_point : 0.0;

        fwrite(&ok,4,1,o);
        fwrite(&action,4,1,o);
        fwrite(&eND,8,1,o);
        fwrite(&eD,8,1,o);
        fwrite(&tp,8,1,o);

        int32_t b_enabled = ok ? bv.beaver.enabled : 0;
        int32_t b_action = ok ? (int32_t)bv.beaver.action : 0;
        double b_double = ok ? bv.beaver.equity_double : 0.0;
        double b_take = ok ? bv.beaver.equity_take : 0.0;
        double b_beaver = ok ? bv.beaver.equity_beaver : 0.0;
        int32_t b_bv = ok ? bv.beaver.beaver : 0;
        int32_t b_rc = ok ? bv.beaver.raccoon : 0;
        fwrite(&b_enabled,4,1,o);
        fwrite(&b_action,4,1,o);
        fwrite(&b_double,8,1,o);
        fwrite(&b_take,8,1,o);
        fwrite(&b_beaver,8,1,o);
        fwrite(&b_bv,4,1,o);
        fwrite(&b_rc,4,1,o);
    }
    fclose(o); fclose(f);
    fprintf(stderr, "%d cases done\n", count);
    return 0;
}
