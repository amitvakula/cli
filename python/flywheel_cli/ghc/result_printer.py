import io


class ResultTreePrinter(object):
    def __init__(self,
                 result,
                 prefix_first='┬──',
                 prefix_middle='├──',
                 prefix_last='└──',
                 parent_prefix_last='    ',
                 parent_prefix_middle='│   '):
        self.result = result
        self.prefix_first = prefix_first
        self.prefix_middle = prefix_middle
        self.prefix_last = prefix_last
        self.parent_prefix_last = parent_prefix_last
        self.parent_prefix_middle = parent_prefix_middle

    def generate_tree(self, skip_series=False):
        with io.StringIO("some initial text data") as f:
            for i, study in enumerate(self.result['Studies']):
                if i == 0:
                    _prefix = self.prefix_first
                elif i == len(self.result['Studies'])-1:
                    _prefix = self.prefix_last
                else:
                    _prefix = self.prefix_middle
                f.write('%s %s (%s, %s, %d series)\n' % (
                _prefix, study['StudyInstanceUID'], study['StudyDate'], study['StudyDescription'] or 'no description',
                study['NumberOfSeries']))

                if skip_series:
                    continue

                for j, v in enumerate(study['series']):
                    _prefix = (self.prefix_last
                               if j + 1 == len(study['series'])
                               else self.prefix_middle)
                    parts = ['%s %s (%s, %s instances)\n' % (
                    _prefix, v['SeriesInstanceUID'], v['SeriesDescription'], v['NumberOfInstances'])]
                    if i + 1 == len(self.result['Studies']):
                        parts.append(self.parent_prefix_last)
                    else:
                        parts.append(self.parent_prefix_middle)

                    f.write(''.join(reversed(parts)))

            return f.getvalue()
