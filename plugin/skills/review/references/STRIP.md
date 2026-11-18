# Strip

Every text fetched from the forge, the description, each comment, the spec issue
and its comments, and the job names, is saved raw and stripped into its own
file by this pass, and a brief quotes only the stripped file:

```sh
perl -CSD -0777 -ne '
my $a = q{(?:[^<>"\x27]|"[^"]*"|\x27[^\x27]*\x27)*};
my $none = q{(?:display\s*:\s*none|visibility\s*:\s*hidden)};
my $hide = qq{(?:\\shidden(?=[\\s/>=])|\\sstyle\\s*=\\s*(?:"[^"]*$none\[^"]*"|\\x27[^\\x27]*$none\[^\\x27]*\\x27|$none))};
s/<!--.*?(?:-->|\z)//gs;
s/<(script|style|noscript|template|textarea)\b$a>.*?(?:<\/\1\s*>|\z)//gis;
while (/<(\w+)\b$a$hide$a>/gi) {
  my ($name, $start, $open, $depth) = ($1, $-[0], $+[0], substr($&, -2) eq "/>" ? 0 : 1);
  while ($depth && /<(\/?)\Q$name\E\b$a>/gi) { $depth += $1 ? -1 : 1 }
  my $end = $depth ? $open : pos;
  substr($_, $start, $end - $start, "");
  pos = $start;
}
s/!\[[^\]]*\](?:\([^)]*\)|\[[^\]]*\])/[image]/g;
s/^[ \t]*\[[^\]]*\]:[ \t]*\S.*$//mg;
s/&#(?:x[0-9a-f]+|\d+);//gi;
s/<\/?[a-z][\w-]*$a\/?>//gi;
s/[\x{00AD}\x{061C}\x{180E}\x{200B}-\x{200F}\x{202A}-\x{202E}\x{2060}-\x{2064}\x{2066}-\x{2069}\x{FE00}-\x{FE0F}\x{FEFF}\x{E0000}-\x{E007F}\x{E0100}-\x{E01EF}]//g;
print' <raw> > <stripped>
```

It removes HTML comments; `script`, `style`, `noscript`, `template` and
`textarea` elements; elements marked `hidden` or styled `display: none` or
`visibility: hidden`, with all they hold, or only the tag where it has no
closing tag; images, link-reference lines such as `[//]: # (…)`, numeric
character references, the remaining tags with their text kept, and invisible
characters. A raw file that matched `hidden`, `display: none` or `<!--` is a
notice, naming the file: the pass is a filter, and the approvals in steps 9 and
12 are what limits what is written.
