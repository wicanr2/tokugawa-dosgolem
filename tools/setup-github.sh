#!/bin/sh
# 在主機終端執行：sh tools/setup-github.sh
# --prepare 只在 Docker 產生待送出的本文，不存取 GitHub。
# GitHub CLI 契約：
# https://cli.github.com/manual/gh_repo_create
# https://cli.github.com/manual/gh_issue_create
# https://cli.github.com/manual/gh_api
set -eu

case "${1-}" in
    ""|--prepare) ;;
    *) printf '%s\n' '用法：sh tools/setup-github.sh [--prepare]' >&2; exit 2 ;;
esac
test "$#" -le 1

project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
gh=/home/anr2/.local/bin/gh
test -f "$project/WORKLIST.md"
test -d "$project/workplace"
test -d "$project/tools"

# 只把 Docker 用於本文擷取。GitHub 登入與 API 一律由下方主機 gh 執行。
timeout 40s docker run --rm --interactive --name tokugawa-gh-prepare --network none \
    --memory 256m --cpus 1 --pids-limit 32 --user "$(id -u):$(id -g)" \
    --mount "type=bind,src=$project,dst=/project,readonly" \
    --mount "type=bind,src=$project/workplace,dst=/output" \
    python@sha256:5f55cdf0c5d9dc1a415637a5ccc4a9e18663ad203673173b8cda8f8dcacef689 \
    timeout 30s python3 - /output <<'PY'
import json
import os
from pathlib import Path
import re
import sys

source = Path("/project/WORKLIST.md").read_text(encoding="utf-8")
sections = re.findall(
    r"^### (CD-TRACE|CDDA|DESKTOP|I18N|HD|ACCEPTANCE)：([^\n]+)\n(.*?)(?=^### |\Z)",
    source, re.MULTILINE | re.DOTALL,
)
expected = ["CD-TRACE", "CDDA", "DESKTOP", "I18N", "HD", "ACCEPTANCE"]
if [key for key, _, _ in sections] != expected:
    raise SystemExit("Issue 草稿缺少、重複或順序錯誤；未存取 GitHub。")
output_root = Path(sys.argv[1])
if output_root.stat().st_uid != os.getuid():
    raise SystemExit("輸出目錄擁有者不同；停止。")
output = output_root / "github-setup"
if output.is_symlink():
    raise SystemExit("輸出目錄不能是符號連結。")
output.mkdir(exist_ok=True)
if output.stat().st_uid != os.getuid():
    raise SystemExit("輸出目錄擁有者不同；停止。")
for key, title, body in sections:
    if not body.strip() or any(c in title for c in '\t\r"\\'):
        raise SystemExit("Issue 本文或標題不符合擷取契約。")
    for name, text in (
        (key + ".md", body.strip() + "\n"),
        (key + ".title", title + "\n"),
        (key + ".query", '.[] | select(.pull_request == null and (.title == '
         + json.dumps(title, ensure_ascii=False)
         + ' or ((.body // "") | contains("<!-- tokugawa-dosgolem:'
         + key + ' -->")))) | .html_url\n'),
    ):
        p = output / name
        if p.is_symlink() or (p.exists() and p.stat().st_uid != os.getuid()):
            raise SystemExit("拒絕寫入非本人檔案或符號連結：" + str(p))
        p.write_text(text, encoding="utf-8")
        if p.stat().st_uid != os.getuid():
            raise SystemExit("輸出擁有權不符：" + str(p))
print("已擷取六筆 Issue 本文；尚未存取 GitHub。")
PY

output=$project/workplace/github-setup
if test "${1-}" = --prepare; then
    printf '本文目錄：%s\n' "$output"
    exit 0
fi

test -x "$gh"
export GH_HOST=github.com
export GH_PROMPT_DISABLED=1
"$gh" auth status --hostname github.com
owner=$("$gh" api user --jq .login)
case "$owner" in
    ""|*[!a-zA-Z0-9-]*) printf '%s\n' 'GitHub 帳號格式不符；停止。' >&2; exit 1 ;;
esac
repo=$owner/tokugawa-dosgolem

# 列舉所有頁，失敗即停止。不能把 API 失敗當成 repository 不存在。
visibility=$("$gh" api --paginate '/user/repos?type=owner&per_page=100' \
    --jq ".[] | select(.full_name == \"$repo\") | .private")
case "$visibility" in
    "") "$gh" repo create "$repo" --private \
        --description '將 DOS 原版《德川家康》整合到 dosgolem，支援桌面 CD 音樂、四語顯示與可切換 HD 素材包。' ;;
    true) printf '沿用私人儲存庫：%s\n' "$repo" ;;
    *) printf '%s\n' '同名儲存庫不是 private；停止，不自動改動。' >&2; exit 1 ;;
esac
test "$("$gh" api "repos/$repo" --jq .private)" = true
repo_url=$("$gh" api "repos/$repo" --jq .html_url)
printf '私人儲存庫：%s\n' "$repo_url"

cd_trace_url=
cdda_url=
desktop_url=
i18n_url=
hd_url=
for key in CD-TRACE CDDA DESKTOP I18N HD ACCEPTANCE; do
    IFS= read -r title < "$output/$key.title"
    IFS= read -r query < "$output/$key.query"
    issue_url=$("$gh" api --paginate "repos/$repo/issues?state=all&per_page=100" --jq "$query")
    case "$issue_url" in
        *'
'*) printf '找到多筆同題 Issue：%s；停止，請核對。\n' "$key" >&2; exit 1 ;;
    esac
    if test -z "$issue_url"; then
        # 此檔只供本次送出，保留完整換行與已建立的相依 URL。
        body=$output/$key.submit.md
        test ! -L "$body"
        if test -e "$body"; then test -O "$body"; fi
        {
            printf '<!-- tokugawa-dosgolem:%s -->\n\n' "$key"
            case "$key" in
                CDDA) printf '相依 Issue：%s\n\n' "$cd_trace_url" ;;
                DESKTOP) printf '音訊驗收相依 Issue：%s\n\n' "$cdda_url" ;;
                I18N|HD) printf '相依 Issue：%s\n\n' "$desktop_url" ;;
                ACCEPTANCE) printf '相依 Issues：\n\n- %s\n- %s\n- %s\n- %s\n- %s\n\n' \
                    "$cd_trace_url" "$cdda_url" "$desktop_url" "$i18n_url" "$hd_url" ;;
            esac
            cat "$output/$key.md"
        } > "$body"
        issue_url=$("$gh" issue create --repo "$repo" --title "$title" --body-file "$body")
    fi
    case "$issue_url" in
        "https://github.com/$repo/issues/"[0-9]*) ;;
        *) printf 'Issue URL 格式不符：%s；停止。\n' "$key" >&2; exit 1 ;;
    esac
    case "$key" in
        CD-TRACE) cd_trace_url=$issue_url ;;
        CDDA) cdda_url=$issue_url ;;
        DESKTOP) desktop_url=$issue_url ;;
        I18N) i18n_url=$issue_url ;;
        HD) hd_url=$issue_url ;;
    esac
    printf '%s：%s\n' "$key" "$issue_url"
done
printf '%s\n' '私人 repo 與六筆 Issues 已核對；可開始 CD-TRACE。'
