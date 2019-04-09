"""Provide helpers for printing to the console"""
import os
import sty


NO_COLOR=None


def print_table(out, table):
    # Calculate max width
    widths = [0] * len(table[0])
    for row in table:
        for i, col in enumerate(row):
            widths[i] = max(widths[i], len(col))

    for row in table:
        for i, col in enumerate(row):
            print(col.ljust(widths[i]), end=' ', file=out)
        print(file=out)


def green_bold(s):
    return style(s, sty.fg.green + sty.ef.bold, sty.rs.bold_dim + sty.rs.fg)


def blue_bold(s):
    return style(s, sty.fg.blue + sty.ef.bold, sty.rs.bold_dim + sty.rs.fg)


def style(s, sty_set, sty_unset):
    global NO_COLOR

    if NO_COLOR is None:
        NO_COLOR = 'NO_COLOR' in os.environ

    if NO_COLOR:
        return s

    return sty_set + s + sty_unset
