/* reference_alternate.c -- the C reference's answer for the network gammonNet
 * publishes beside the default one (`alternate_network`), on the feature
 * vectors of reference16.bin. Writes outputs only: magic "GNRO", count,
 * outputs, then count x 5 float32, little-endian. See README.md. */
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#include "gn_infer.h"

int main(int argc, char **argv)
{
    if (argc != 4) {
        fprintf(stderr, "usage: %s reference16.bin network.bin16 out.bin\n", argv[0]);
        return 2;
    }
    FILE *f = fopen(argv[1], "rb");
    if (!f) { perror(argv[1]); return 1; }
    uint32_t head[4];
    if (fread(head, 4, 4, f) != 4 || head[0] != 0x46524E47u) { fprintf(stderr, "bad reference\n"); return 1; }
    const int count = (int)head[1], nf = (int)head[2], no = (int)head[3];
    float *feat = malloc(sizeof(float) * (size_t)count * nf);
    if (!feat || fread(feat, sizeof(float), (size_t)count * nf, f) != (size_t)count * nf) { fprintf(stderr, "short reference\n"); return 1; }
    fclose(f);
    GnNetwork *net = gn_network_load(argv[2]);
    if (!net) { fprintf(stderr, "model load failed\n"); return 1; }
    FILE *o = fopen(argv[3], "wb");
    if (!o) { perror(argv[3]); return 1; }
    const uint32_t out_head[3] = {0x4F524E47u, (uint32_t)count, (uint32_t)no};
    fwrite(out_head, 4, 3, o);
    for (int i = 0; i < count; i++) {
        float probs[GN_NUM_OUTPUTS];
        if (gn_evaluate_features(net, feat + (size_t)i * nf, probs) != 0) { fprintf(stderr, "eval %d failed\n", i); return 1; }
        fwrite(probs, sizeof(float), (size_t)no, o);
    }
    fclose(o);
    gn_network_free(net);
    return 0;
}
