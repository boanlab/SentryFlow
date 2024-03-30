from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class APIClassificationRequest(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, path: _Optional[_Iterable[str]] = ...) -> None: ...

class APIClassificationResponse(_message.Message):
    __slots__ = ("merged", "fields")
    MERGED_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    merged: str
    fields: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, merged: _Optional[str] = ..., fields: _Optional[_Iterable[str]] = ...) -> None: ...
