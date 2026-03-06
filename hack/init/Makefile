BUTANE ?= butane

%.ign: %.bu
	$(BUTANE) --strict --files-dir ../.. < $< > $@

config.ign: base.ign syringe.ign users.ign
	$(BUTANE) --strict --files-dir . < config.bu > $@

.PHONY: clean
clean:
	rm -f *.ign
