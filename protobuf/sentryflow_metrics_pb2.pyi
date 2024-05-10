from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class APIClassifierRequest(_message.Message):
    __slots__ = ("API",)
    API_FIELD_NUMBER: _ClassVar[int]
    API: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, API: _Optional[_Iterable[str]] = ...) -> None: ...

class APIClassifierResponse(_message.Message):
    __slots__ = ("APIs",)
    class APIsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    APIS_FIELD_NUMBER: _ClassVar[int]
    APIs: _containers.ScalarMap[str, int]
    def __init__(self, APIs: _Optional[_Mapping[str, int]] = ...) -> None: ...
