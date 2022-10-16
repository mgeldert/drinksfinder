#!/usr/bin/env python3

import json
import re
from flask import Flask, request, jsonify


app = Flask(__name__)


def sanitize_field(header : str):
    h = header.replace("\"", "")
    if h == "tags\n":
        return h.strip()
    return h


class DrinkingEstablishmentList(list):
    _RATING_CRITERIA = ["beer", "atmosphere", "amenities", "value"]

    def __init__(self, datafile_name: str):
        with open(datafile_name) as f:
            data = f.readlines()
        field_names = [sanitize_field(header) for header in data[0].split(",")]
        for establishment in data[1:]:
            self.append(DrinkingEstablishment(field_names, establishment))
        self._populate_caches()

    def get_list(self, ordered_by: str=None, with_tags: list=None, near: tuple=None):
        base_list = self[:]
        if ordered_by is not None:
            if ordered_by not in self._RATING_CRITERIA:
                raise Exception(f"Invalid rating criteria '{ordered_by}'")
            base_list = getattr(self, f"_sort_by_{ordered_by}_cache")[:]
        if with_tags is not None:
            for entry in base_list[:]:
                for tag in with_tags:
                    if tag not in entry.tags:
                        base_list.remove(entry)
                        break
        if near is not None:
            proximity_list = []
            for entry in base_list:
                lng_diff = abs(near[0] - float(entry.lng))
                lat_diff = abs(near[1] - float(entry.lat))
                distance = (lat_diff**2 + lng_diff**2)**0.5
                proximity_list.append((entry, distance))
            proximity_list = sorted(proximity_list, key=lambda x: x[1])
            base_list = [p[0] for p in proximity_list]
        return base_list
            
    def _populate_caches(self):
        """
        This is to simulate a caching layer between the API server and
        the datastore.
        """
        for rating in self._RATING_CRITERIA:
            setattr(self,
                    f"_sort_by_{rating}_cache",
                    sorted(
                        self,
                        key=lambda row: float(getattr(row, f"stars_{rating}")),
                        reverse=True
                    )
            )


class DrinkingEstablishment:
    def __init__(self, fields, row):
        values = self._get_fields(row)
        for index, field_name in enumerate(fields):
            setattr(self, field_name, values[index])

    def _get_fields(self, row: str) -> list:
        """
        Split a row from the CSV into a list of elements.
        N.B. Can't just split on comma because some fields contain commas.

        Potential enhancements - could add a field containing the average score
                                 across star ratings for an "overall" rating
                               - could drop the records where the category
                                 is "Closed venues", but leaving that for the
                                 client to decide
        """
        row = row.strip()
        fields = []
        while True:
            # Match the first quote-wrapped field and the remainder of the
            # string minus leading comma and whitespace
            result = re.search('"(.*?)"\s*,\s*(.*)', row)
            try:
                fields.append(result.group(1))
            except AttributeError:
                # Didn't match the regex, so we're on the last field, "tags",
                # which we want to convert from comma-separated string to a
                # list.
                fields.append([sanitize_field(f) for f in row.split(",")])
                break
            row = result.group(2)
        return fields


@app.route("/ready", methods=["GET"])
def ready():
    return jsonify({"status": "ok"})


@app.route("/", methods=["PUT"])
def data_access():
    search_params = {}
    body = request.get_json()
    try:
        search_params['near'] = (body['longlat'][0], body['longlat'][1])
    except KeyError:
        pass
    try:
        search_params['ordered_by'] = body['order']
    except KeyError:
        pass
    try:
        search_params['with_tags'] = body['tags']
    except KeyError:
        pass
    return_data = [
        vars(est) for est in drinking_establishments.get_list(**search_params)
    ]
    return jsonify(return_data)


def main():
    global drinking_establishments
    drinking_establishments = DrinkingEstablishmentList("leedsbeerquest.csv")
    app.run(host="0.0.0.0", port=8080)


if __name__ == "__main__":
    main()
