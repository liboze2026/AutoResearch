"""Method registry namespace for phase4 retrieval methods."""
from methods.dummy_retrieval import DummyRetrievalMethod
from methods.page_lexical_retrieval import PageLexicalRetrievalMethod

__all__ = ["DummyRetrievalMethod", "PageLexicalRetrievalMethod"]
