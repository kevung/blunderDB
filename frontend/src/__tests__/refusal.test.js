/**
 * refusal.test.js — the refusals of `like` reach the user in the user's language, read from the
 * code the backend heads them with, never from the English sentence after it.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';

const { GetLikeMaxOrder } = vi.hoisted(() => ({ GetLikeMaxOrder: vi.fn() }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetLikeMaxOrder }));

import { refusalReason } from '../utils/refusal.js';
import { tMsg } from '../i18n';

const PREFIX = 'storage: invalid argument: ';

beforeEach(() => {
    GetLikeMaxOrder.mockReset().mockResolvedValue(2);
});

describe('refusalReason', () => {
    test('another failure is no refusal', async () => {
        expect(await refusalReason(new Error('disk full'))).toBe('');
    });

    test('`like` outside a ranking', async () => {
        const err = new Error(`${PREFIX}refused query: like-unranked: \`like\` is served only by a ranked search`);
        expect(await refusalReason(err)).toEqual(tMsg('similar.unrankedRefused'));
    });

    test('a starred `like`, named', async () => {
        const err = `${PREFIX}refused query: like-starred: like42<1*: the starred \`like\` is no longer served`;
        expect(await refusalReason(err)).toEqual(tMsg('similar.widenedRefused', { token: 'like42<1*' }));
    });

    test('an order beyond the maximum, the maximum read from Go', async () => {
        GetLikeMaxOrder.mockResolvedValue(3);
        const err = `${PREFIX}like-order: a search within 4 plays is not served: at most 3`;
        expect(await refusalReason(err)).toEqual(tMsg('similar.orderRefused', { max: 3 }));
    });

    test('without the maximum, the backend’s own words', async () => {
        GetLikeMaxOrder.mockRejectedValue(new Error('no config'));
        const reason = 'like-order: a search within 4 plays is not served: at most 2';
        expect(await refusalReason(PREFIX + reason)).toBe(reason);
    });

    test('an uncoded refusal is shown as the backend worded it', async () => {
        expect(await refusalReason(`${PREFIX}collection 3 is not living: nothing to freeze`)).toBe('collection 3 is not living: nothing to freeze');
    });
});
