QT -= gui
QT += core

CONFIG += c++11 console
CONFIG -= app_bundle

TARGET = SimTargetHit
TEMPLATE = app

SOURCES += main.cpp

HEADERS += httplib.h json.hpp

DESTDIR = bin

