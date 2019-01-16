from collections import (
    namedtuple,
)
import re

from dateutil import (
    parser,
)

from .logs import (
    convert_type,
    map_log_enum_pattern,
)


class NoMatchingPattern(Exception):
    pass


class ParsingError(Exception):
    pass


Event = namedtuple('Event', ['time', 'log_type', 'logger_name', 'event_type', 'params'])


_map_log_enum_pats = {
    key: re.compile(value)
    for key, value in map_log_enum_pattern.items()
}


def parse_line(line):
    """Try over all patterns. Parse the line with the first matching pattern to a `Event`.
        Returns `None` when no pattern matches.
    """
    for log_enum, pat in _map_log_enum_pats.items():
        match = pat.search(line)
        # only return if a event is found
        if match is not None:
            matched_fields = match.groups()
            try:
                log_time = parser.parse(matched_fields[0])
            except ValueError:
                raise ParsingError("malform log_time: {!r}".format(matched_fields[0]))
            event = Event(
                time=log_time,
                log_type=matched_fields[1],
                logger_name=matched_fields[2],
                event_type=log_enum,
                # TODO: parse the params with `convert_type`
                # params=convert_type(matched_fields[3:], log_enum),
                params=matched_fields[3:],
            )
            return event
    raise NoMatchingPattern("line={!r}".format(line))
