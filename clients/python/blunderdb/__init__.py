"""A minimal Python client for the blunderDB engine API (#289).

    from blunderdb import Client

    api = Client("http://127.0.0.1:8080", tenant=1)
    counts = api.metadata_counts()
    for position in api.positions_list({"limit": 10}):
        print(position["id"])

The method surface is generated from the daemon's own route table, so it
cannot fall behind it; the transport is hand-written. See README.md.
"""

from ._generated import GeneratedAPI
from .client import APIError, BaseClient

__all__ = ["Client", "APIError", "BaseClient"]


class Client(GeneratedAPI):
    """The whole API: the generated methods over the hand-written transport."""
