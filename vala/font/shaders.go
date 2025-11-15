package font

// SDFVertexShader renders textquads with position and UV coordinates
const SDFVertexShader = `
#version 450

// Per-vertex attributes
layout(location = 0) in vec2 inPosition;  // Screen-space position
layout(location = 1) in vec2 inTexCoord;  // UV coordinates in atlas

// Per-instance/uniform data (color, transform, etc.)
layout(push_constant) uniform PushConstants {
    vec2 screenSize;   // Screen dimensions for NDC conversion
    vec4 textColor;    // Text color (RGBA)
} push;

// Output to fragment shader
layout(location = 0) out vec2 fragTexCoord;
layout(location = 1) out vec4 fragColor;

void main() {
    // Convert screen-space pixel coordinates to NDC (-1 to 1)
    vec2 ndc = (inPosition / push.screenSize) * 2.0 - 1.0;
    // Note: Y-flip removed - screen Y=0 is top, NDC Y=-1 is top (Vulkan convention)

    gl_Position = vec4(ndc, 0.0, 1.0);
    fragTexCoord = inTexCoord;
    fragColor = push.textColor;
}
`

// SDFFragmentShader renders SDF glyphs with smooth anti-aliasing
const SDFFragmentShader = `
#version 450

// Input from vertex shader
layout(location = 0) in vec2 fragTexCoord;
layout(location = 1) in vec4 fragColor;

// SDF atlas texture
layout(binding = 0) uniform sampler2D sdfAtlas;

// Output
layout(location = 0) out vec4 outColor;

void main() {
    // DEBUG: Output solid color to verify quads are rendering
    //outColor = fragColor;

    /* ORIGINAL SDF RENDERING (disabled for debugging): */
    // Sample SDF value (0-1, where 0.5 is the edge)
    float distance = texture(sdfAtlas, fragTexCoord).r;

    // Convert to signed distance (negative = outside, positive = inside)
    float signedDist = distance - 0.5;

    // Calculate alpha based on distance with smooth anti-aliasing
    // fwidth gives us the rate of change per pixel for automatic AA
    float alpha = smoothstep(-fwidth(signedDist), fwidth(signedDist), signedDist);

    // Output color with calculated alpha
    // alpha is 0 outside, 1 inside after smoothstep
    outColor = vec4(fragColor.rgb, fragColor.a * alpha);

    // Discard fully transparent pixels (optimization)
    if (outColor.a < 0.01) {
        discard;
    }
    /**/
}
`
