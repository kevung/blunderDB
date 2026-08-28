/* Produces the gold file for blunderDB's Go port of gammonNet's search.
 * Reads testdata/search_corpus.bin, writes search_gold.bin.
 * Canonical configuration: prune k=12, filter {0,1,3,5,5}. */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "gn_search.h"
#include "gn_infer.h"
#include "gn_rules.h"

#define TOPK 5

int main(int argc, char **argv) {
    if (argc < 5) { fprintf(stderr, "usage: gold corpus.bin model.bin prune.bin out.bin\n"); return 2; }
    FILE *f = fopen(argv[1], "rb");
    if (!f) { perror("corpus"); return 1; }
    char magic[4]; int count;
    if (fread(magic,1,4,f)!=4 || memcmp(magic,"GNCP",4) || fread(&count,4,1,f)!=1) {
        fprintf(stderr,"bad corpus\n"); return 1; }

    GnNetwork *net = gn_network_load(argv[2]);
    GnNetwork *prune = gn_network_load(argv[3]);
    if (!net || !prune) { fprintf(stderr,"model load failed\n"); return 1; }

    FILE *o = fopen(argv[4], "wb");
    fwrite("GNGD",1,4,o); fwrite(&count,4,1,o);

    GnCandidate *cands = malloc(sizeof(GnCandidate)*2048);
    for (int i = 0; i < count; i++) {
        unsigned char buf[32];
        if (fread(buf,1,32,f)!=32) { fprintf(stderr,"short corpus at %d\n",i); return 1; }
        GnPosition pos;
        for (int j=0;j<24;j++) pos.points[j] = (signed char)buf[j];
        pos.bar[0]=buf[24]; pos.bar[1]=buf[25];
        pos.off[0]=buf[26]; pos.off[1]=buf[27]; pos.turn=buf[28];
        int d1=buf[29], d2=buf[30], ply=buf[31];

        GnSearchConfig cfg = gn_search_config(ply);
        cfg.filter[0]=0; cfg.filter[1]=1; cfg.filter[2]=3; cfg.filter[3]=5; cfg.filter[4]=5;
        gn_search_use_prune(&cfg, prune, 12);

        int n = gn_search_plays(net, &pos, d1, d2, &cfg, cands, 2048);
        if (n < 0) { fprintf(stderr,"search refused case %d\n", i); return 1; }
        int k = n < TOPK ? n : TOPK;
        int32_t n32 = n, k32 = k;
        fwrite(&n32,4,1,o); fwrite(&k32,4,1,o);
        for (int c = 0; c < k; c++) {
            const GnPosition *r = &cands[c].play.result;
            unsigned char rb[29];
            for (int j=0;j<24;j++) rb[j]=(unsigned char)r->points[j];
            rb[24]=r->bar[0]; rb[25]=r->bar[1]; rb[26]=r->off[0]; rb[27]=r->off[1]; rb[28]=r->turn;
            fwrite(rb,1,29,o);
            double e = cands[c].equity;
            fwrite(&e,8,1,o);
        }
        if ((i+1)%10==0) { fprintf(stderr,"\r%d/%d", i+1, count); fflush(stderr); }
    }
    fprintf(stderr,"\r%d/%d done\n", count, count);
    fclose(o); fclose(f);
    return 0;
}
