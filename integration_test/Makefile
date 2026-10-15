POSTGRES_TARGETS := pg12 pg13 pg14 pg15 pg16 pg17
SCRIPT_TARGETS := reload guided-setup installer otel-vector

TARGETS := $(POSTGRES_TARGETS) $(SCRIPT_TARGETS)

# Citus doesn't release ARM images, thus skip on ARM
ifeq ($(findstring $(shell uname -m),arm64 aarch64),)
	TARGETS += citus
endif

.PHONY: all $(POSTGRES_TARGETS) citus $(SCRIPT_TARGETS)

all: $(TARGETS)

$(POSTGRES_TARGETS) citus:
	./postgres.sh $@

$(SCRIPT_TARGETS):
	./$@.sh
