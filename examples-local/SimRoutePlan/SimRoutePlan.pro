QT -= gui

CONFIG += c++11 console
CONFIG -= app_bundle

DEFINES += QT_DEPRECATED_WARNINGS

MOC_DIR=build
OBJECTS_DIR=build
UI_DIR=build
RCC_DIR=build
DESTDIR=bin

SOURCES += \
    main.cpp

HEADERS += \
    httplib.h \
    json.hpp
