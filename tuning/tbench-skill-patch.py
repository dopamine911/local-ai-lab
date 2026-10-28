#!/usr/bin/env python3
"""Let Terminal-Bench-Local's prompt hook add an instruction file (a skill) to the agent prompt.

Applied by `./tbench setup`. Edits terminal-bench/compat/harbor_prompt/sitecustomize.py so that,
when TBENCH_EXTRA_INSTRUCTIONS names a file, its text (frontmatter stripped) is placed before
"Task Description:" in Terminus-2's prompt, after the runner's own completion instruction.
Without the variable, behaviour is unchanged. Safe to run more than once.
"""
import pathlib
import sys

hook = pathlib.Path(sys.argv[1]) / "compat" / "harbor_prompt" / "sitecustomize.py"
src = hook.read_text()
MARK = "# local-ai-lab: extra instructions"
if MARK in src:
    print("skill patch already applied")
    sys.exit(0)

anchor = (
    "            self._prompt_template = self._prompt_template.replace(\n"
    "                marker, _completion_instruction + marker, 1\n"
    "            )\n"
)
if anchor not in src:
    sys.exit("Terminal-Bench's prompt hook changed; the skill patch needs updating")

addition = anchor + f"""            {MARK}
            import os, re
            _extra = os.environ.get("TBENCH_EXTRA_INSTRUCTIONS")
            if _extra:
                _text = re.sub(r"\\A---\\n.*?\\n---\\n", "", open(_extra).read(), count=1, flags=re.S).strip()
                _text = _text.replace("{{", "{{{{").replace("}}", "}}}}")  # template is filled with str.format
                self._prompt_template = self._prompt_template.replace(
                    marker, "Operating instructions:\\n\\n" + _text + "\\n\\n" + marker, 1
                )
"""
hook.write_text(src.replace(anchor, addition, 1))
print("skill patch applied to", hook)
