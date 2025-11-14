#pragma once

#include <QtCore/QDateTime>
#include <QtCore/QJsonArray>
#include <QtCore/QJsonDocument>
#include <QtCore/QJsonObject>
#include <QtCore/QRandomGenerator>
#include <QtCore/QString>
#include <QtCore/QVector>
#include <cmath>
#include <stdexcept>

struct GeoPoint {
    double lat = 0.0;
    double lon = 0.0;
};

struct TingState {
    QString id;
    QString name;
    GeoPoint position;
    double speedMps = 8.0;
    double sonarRange = 80.0;
    double suspectProb = 0.4;
    double confirmProb = 0.6;
    double elapsedSeconds = 0.0;
};

struct WorkZone {
    QString id;
    int index = 0;
    GeoPoint topLeft;
    GeoPoint bottomRight;
};

struct MineInfo {
    QString id;
    GeoPoint position;
    QString status;
    QString assignedTing;
};

inline GeoPoint geoPointFromJson(const QJsonObject &obj, const QString &name) {
    if (!obj.contains("lat") || !obj.contains("lon")) {
        throw std::runtime_error(QString("%1 缺少 lat 或 lon 字段").arg(name).toStdString());
    }
    GeoPoint p;
    p.lat = obj.value("lat").toDouble();
    p.lon = obj.value("lon").toDouble();
    return p;
}

inline QJsonObject geoPointToJson(const GeoPoint &point) {
    QJsonObject obj;
    obj.insert("lat", point.lat);
    obj.insert("lon", point.lon);
    return obj;
}

inline double deg2rad(double deg) {
    return deg * M_PI / 180.0;
}

inline double haversineDistanceMeters(const GeoPoint &a, const GeoPoint &b) {
    static constexpr double EarthRadius = 6371000.0;  // 米
    const double lat1 = deg2rad(a.lat);
    const double lat2 = deg2rad(b.lat);
    const double dLat = deg2rad(b.lat - a.lat);
    const double dLon = deg2rad(b.lon - a.lon);

    const double sinLat = std::sin(dLat / 2.0);
    const double sinLon = std::sin(dLon / 2.0);
    const double h = sinLat * sinLat + std::cos(lat1) * std::cos(lat2) * sinLon * sinLon;
    const double c = 2.0 * std::atan2(std::sqrt(h), std::sqrt(1.0 - h));
    return EarthRadius * c;
}

inline double estimateTravelTimeSeconds(const GeoPoint &from, const GeoPoint &to, double speedMps) {
    if (speedMps <= 0.0) {
        return 0.0;
    }
    return haversineDistanceMeters(from, to) / speedMps;
}

inline GeoPoint interpolate(const GeoPoint &from, const GeoPoint &to, double t) {
    GeoPoint result;
    result.lat = from.lat + (to.lat - from.lat) * t;
    result.lon = from.lon + (to.lon - from.lon) * t;
    return result;
}

inline void appendLinearTrack(QJsonArray &tracks,
                              const QString &tingId,
                              const QString &phase,
                              const GeoPoint &start,
                              const GeoPoint &end,
                              double speed,
                              double &elapsedSeconds,
                              const QDateTime &phaseStart,
                              double timeStepSeconds = 0.5) {
    const double travelTime = estimateTravelTimeSeconds(start, end, speed);
    int steps = static_cast<int>(std::ceil(travelTime / timeStepSeconds));
    if (steps < 1) {
        steps = 1;
    }

    for (int i = 0; i <= steps; ++i) {
        const double ratio = static_cast<double>(i) / static_cast<double>(steps);
        const GeoPoint point = interpolate(start, end, ratio);
        const double timestampOffset = elapsedSeconds + ratio * travelTime;
        const QDateTime timestamp = phaseStart.addMSecs(static_cast<qint64>(timestampOffset * 1000.0));

        QJsonObject trackPoint;
        trackPoint.insert("ting_id", tingId);
        trackPoint.insert("phase", phase);
        trackPoint.insert("timestamp", timestamp.toString(Qt::ISODate));
        trackPoint.insert("position", geoPointToJson(point));
        tracks.append(trackPoint);
    }

    elapsedSeconds += travelTime;
}

inline void appendDwellTrack(QJsonArray &tracks,
                             const QString &tingId,
                             const QString &phase,
                             const GeoPoint &position,
                             double &elapsedSeconds,
                             const QDateTime &phaseStart,
                             double dwellSeconds,
                             int steps = 10) {
    if (dwellSeconds <= 0.0) {
        return;
    }
    if (steps < 1) {
        steps = 1;
    }
    const double stepSeconds = dwellSeconds / static_cast<double>(steps);
    for (int i = 1; i <= steps; ++i) {
        elapsedSeconds += stepSeconds;
        const QDateTime timestamp =
            phaseStart.addMSecs(static_cast<qint64>(elapsedSeconds * 1000.0));

        QJsonObject trackPoint;
        trackPoint.insert("ting_id", tingId);
        trackPoint.insert("phase", phase);
        trackPoint.insert("timestamp", timestamp.toString(Qt::ISODate));
        trackPoint.insert("position", geoPointToJson(position));
        tracks.append(trackPoint);
    }
}

inline QJsonArray serializeTings(const QVector<TingState> &tings) {
    QJsonArray array;
    for (const auto &ting : tings) {
        QJsonObject obj;
        obj.insert("id", ting.id);
        obj.insert("name", ting.name);
        obj.insert("position", geoPointToJson(ting.position));
        obj.insert("speed_mps", ting.speedMps);
        obj.insert("sonar_range_m", ting.sonarRange);
        obj.insert("suspect_prob", ting.suspectProb);
        obj.insert("confirm_prob", ting.confirmProb);
        array.append(obj);
    }
    return array;
}

inline QVector<TingState> parseTings(const QJsonArray &array) {
    QVector<TingState> tings;
    tings.reserve(array.size());
    for (const auto &item : array) {
        const QJsonObject obj = item.toObject();
        TingState ting;
        ting.id = obj.value("id").toString();
        ting.name = obj.value("name").toString(ting.id);
        ting.position = geoPointFromJson(obj.value("position").toObject(), "ting.position");
        ting.speedMps = obj.value("speed_mps").toDouble(8.0);
        ting.sonarRange = obj.value("sonar_range_m").toDouble(80.0);
        ting.suspectProb = obj.value("suspect_prob").toDouble(0.4);
        ting.confirmProb = obj.value("confirm_prob").toDouble(0.6);
        ting.elapsedSeconds = 0.0;
        tings.append(ting);
    }
    return tings;
}

inline QVector<WorkZone> parseZones(const QJsonArray &array) {
    QVector<WorkZone> zones;
    zones.reserve(array.size());
    for (const auto &item : array) {
        const QJsonObject obj = item.toObject();
        WorkZone zone;
        zone.id = obj.value("id").toString();
        zone.index = obj.value("index").toInt();
        zone.topLeft = geoPointFromJson(obj.value("top_left").toObject(), "zone.top_left");
        zone.bottomRight = geoPointFromJson(obj.value("bottom_right").toObject(), "zone.bottom_right");
        zones.append(zone);
    }
    return zones;
}

inline QJsonObject mineToJson(const MineInfo &mine) {
    QJsonObject obj;
    obj.insert("id", mine.id);
    obj.insert("position", geoPointToJson(mine.position));
    obj.insert("status", mine.status);
    obj.insert("assigned_ting", mine.assignedTing);
    return obj;
}

inline QVector<MineInfo> parseMines(const QJsonArray &array, const QString &defaultStatus = QString()) {
    QVector<MineInfo> mines;
    mines.reserve(array.size());
    for (const auto &item : array) {
        const QJsonObject obj = item.toObject();
        MineInfo mine;
        mine.id = obj.value("id").toString();
        mine.position = geoPointFromJson(obj.value("position").toObject(), "mine.position");
        mine.status = obj.value("status").toString(defaultStatus);
        mine.assignedTing = obj.value("assigned_ting").toString();
        mines.append(mine);
    }
    return mines;
}

