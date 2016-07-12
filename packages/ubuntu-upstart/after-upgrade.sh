status pganalyze-collector | grep -q running
if [ $? -eq 0 ]; then
  restart -q pganalyze-collector
else
  start -q pganalyze-collector
fi
