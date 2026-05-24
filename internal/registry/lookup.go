package registry

func (r *Registry) Resolve(deviceID string, alias string) (*ResolvedContext, error) {
	device, ok := r.Devices[deviceID]
	if !ok {
		return nil, registryError(ErrDeviceNotFound, "device=%q", deviceID)
	}

	personID := device.DefaultPerson
	profileID := device.DefaultProfile
	if alias != "" {
		binding := device.Aliases[alias]
		if binding.Person != "" {
			personID = binding.Person
		}
		if binding.Profile != "" {
			profileID = binding.Profile
		}
	}

	profile := r.Profiles[profileID]
	model := r.Models[profile.Model]
	backend := r.Backends[model.Backend]

	return &ResolvedContext{
		DeviceID:  deviceID,
		Alias:     alias,
		PersonID:  personID,
		Person:    r.Persons[personID],
		ProfileID: profileID,
		Profile:   profile,
		ModelID:   profile.Model,
		Model:     model,
		BackendID: model.Backend,
		Backend:   backend,
	}, nil
}
