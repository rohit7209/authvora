"""Geolocation utilities for travel risk calculation."""

import math
from datetime import datetime
from typing import Any

# Earth's radius in kilometers
EARTH_RADIUS_KM = 6371.0

# Impossible travel threshold in km/h (faster than commercial flight)
IMPOSSIBLE_TRAVEL_THRESHOLD_KMH = 900.0


def haversine(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
    """
    Calculate the great-circle distance between two points on Earth.

    Uses the Haversine formula to compute distance in kilometers.

    Args:
        lat1: Latitude of first point in degrees
        lon1: Longitude of first point in degrees
        lat2: Latitude of second point in degrees
        lon2: Longitude of second point in degrees

    Returns:
        Distance in kilometers
    """
    lat1_rad = math.radians(lat1)
    lon1_rad = math.radians(lon1)
    lat2_rad = math.radians(lat2)
    lon2_rad = math.radians(lon2)

    dlat = lat2_rad - lat1_rad
    dlon = lon2_rad - lon1_rad

    a = (
        math.sin(dlat / 2) ** 2
        + math.cos(lat1_rad) * math.cos(lat2_rad) * math.sin(dlon / 2) ** 2
    )
    c = 2 * math.asin(math.sqrt(a))

    return EARTH_RADIUS_KM * c


def calculate_travel_speed(
    lat1: float,
    lon1: float,
    time1: datetime | str,
    lat2: float,
    lon2: float,
    time2: datetime | str,
) -> float:
    """
    Calculate travel speed between two locations in km/h.

    Args:
        lat1: Latitude of origin
        lon1: Longitude of origin
        time1: Timestamp of origin (datetime or ISO string)
        lat2: Latitude of destination
        lon2: Longitude of destination
        time2: Timestamp of destination (datetime or ISO string)

    Returns:
        Speed in km/h, or 0.0 if time difference is zero or invalid
    """
    if isinstance(time1, str):
        time1 = datetime.fromisoformat(time1.replace("Z", "+00:00"))
    if isinstance(time2, str):
        time2 = datetime.fromisoformat(time2.replace("Z", "+00:00"))

    time_diff_seconds = (time2 - time1).total_seconds()
    if time_diff_seconds <= 0:
        return 0.0

    distance_km = haversine(lat1, lon1, lat2, lon2)
    time_hours = time_diff_seconds / 3600.0

    if time_hours <= 0:
        return 0.0

    return distance_km / time_hours


def is_impossible_travel(speed_kmh: float) -> bool:
    """
    Check if the given speed indicates impossible travel.

    Uses threshold of 900 km/h (faster than commercial flight).

    Args:
        speed_kmh: Travel speed in km/h

    Returns:
        True if speed exceeds impossible travel threshold
    """
    return speed_kmh >= IMPOSSIBLE_TRAVEL_THRESHOLD_KMH
