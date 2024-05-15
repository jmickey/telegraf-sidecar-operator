package k8s

const (
	/*
	 * Telagraf Sidecar Container Configuration
	 */

	// SidecarCustomImageAnnotation can be used to override
	// the telegraf sidecar image
	SidecarCustomImageAnnotation = Prefix + "/image"

	// SidecarRequestsCPUAnnotation can be used to override the
	// CPU requests of the sidecar container
	SidecarRequestsCPUAnnotation = Prefix + "/requests-cpu"

	// SidecarRequestsMemoryAnnotation can be used to override the
	// memory requests of the sidecar container
	SidecarRequestsMemoryAnnotation = Prefix + "/requests-memory"

	// SidecarLimitsCPUAnnotation can be used to override the
	// CPU limits of the sidecar container
	SidecarLimitsCPUAnnotation = Prefix + "/limits-cpu"

	// SidecarLimitsMemoryAnnotation can be used to override the
	// memory limits of the sidecar container
	SidecarLimitsMemoryAnnotation = Prefix + "/limits-memory"
)
