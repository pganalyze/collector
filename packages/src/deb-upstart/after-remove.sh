status pganalyze-collector | grep -q running
if [ $? -eq 0 ]; then
  stop -q pganalyze-collector
fi
