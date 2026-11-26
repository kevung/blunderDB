// The tag a map computation's events carry. One counter for every map view of the session: a
// view mounted anew starts above every tag handed out before, so an answer to a computation of
// the view it replaced never matches.
let last = 0;

/** A tag no earlier computation carries. */
export function nextMapTag() {
    return ++last;
}
