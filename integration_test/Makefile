POSTGRES_TARGETS := pg12 pg13 pg14 pg15 pg16 pg17 citus
SCRIPT_TARGETS := reload guided-setup installer otel-vector

TARGETS := pg12 pg13 pg14 pg15 pg16 $(SCRIPT_TARGETS)

# Citus doesn't release ARM images, thus skip on ARM
ifeq ($(findstring $(shell uname -m),arm64 aarch64),)
	TARGETS += citus
endif

.PHONY: all $(POSTGRES_TARGETS) $(SCRIPT_TARGETS)

all: $(TARGETS)

$(POSTGRES_TARGETS):
	./postgres.sh $@

$(SCRIPT_TARGETS):
	./$@.sh
