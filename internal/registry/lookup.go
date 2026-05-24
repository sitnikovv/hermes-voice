package registry

func (r *Registry) Resolve(deviceID string, alias string) (*ResolvedContext, error) {
	device, ok := r.Devices[deviceID]
	if !ok {
		return nil, registryError(ErrDeviceNotFound, "device=%q", deviceID)
	}

	personID := device.DefaultPerson
	profileID := device.DefaultProfile
	if alias != "" {
		binding, ok := device.Aliases[alias]
		if !ok {
			return nil, registryError(ErrAliasNotFound, "device=%q alias=%q", deviceID, alias)
		}
		if binding.Person != "" {
			personID = binding.Person
		}
		if binding.Profile != "" {
			profileID = binding.Profile
		}
	}

	if personID == "" {
		return nil, registryError(ErrMissingDefaultPerson, "device=%q alias=%q", deviceID, alias)
	}
	if profileID == "" {
		return nil, registryError(ErrMissingDefaultProfile, "device=%q alias=%q", deviceID, alias)
	}

	person, ok := r.Persons[personID]
	if !ok {
		return nil, registryError(ErrPersonNotFound, "person=%q", personID)
	}
	profile, ok := r.Profiles[profileID]
	if !ok {
		return nil, registryError(ErrProfileNotFound, "profile=%q", profileID)
	}
	model, ok := r.Models[profile.Model]
	if !ok {
		return nil, registryError(ErrModelNotFound, "model=%q profile=%q", profile.Model, profileID)
	}
	backend, ok := r.Backends[model.Backend]
	if !ok {
		return nil, registryError(ErrBackendNotFound, "backend=%q model=%q", model.Backend, profile.Model)
	}

	return &ResolvedContext{
		DeviceID:  deviceID,
		Alias:     alias,
		PersonID:  personID,
		Person:    person,
		ProfileID: profileID,
		Profile:   profile,
		ModelID:   profile.Model,
		Model:     model,
		BackendID: model.Backend,
		Backend:   backend,
	}, nil
}
