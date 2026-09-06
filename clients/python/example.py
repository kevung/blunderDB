#!/usr/bin/env python3
"""What a script actually does with the engine API (#289).

Run a daemon first:

    blunderdb serve --backend sqlite --db your.db --addr 127.0.0.1:8080

then:

    python3 example.py
"""

from blunderdb import APIError, Client


def main() -> int:
    api = Client("http://127.0.0.1:8080", tenant=1)
    if not api.healthy():
        print("The daemon is not answering on /healthz.")
        return 1

    print("counts:", api.metadata_counts())

    # A list endpoint streams: it is iterated, not materialised.
    print("\nfirst positions:")
    for i, position in enumerate(api.positions_list({"limit": 5})):
        print(f"  {position.get('id')}  decision_type={position.get('decision_type')}")
        if i >= 4:
            break

    # An error comes back as its envelope, not as a stack trace.
    try:
        api.positions_load({"id": -1})
    except APIError as err:
        print(f"\nexpected failure: code={err.code} status={err.status}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
