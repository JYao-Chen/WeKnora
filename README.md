# WeKnora review evidence — 2026-10-10

Before: unmodified official main `0d39af12`. After: local integration of PRs #4034, #4036–#4038, #4040–#4045, including review corrections. These are combined validation screenshots, not isolated PR previews.

Phone: Chromium device emulation, isMobile=true, hasTouch=true, 390px. `default` does not preset sidebar_collapsed; `collapsed` explicitly sets it. Desktop: 1440px, normal expanded-sidebar defaults. The shared shell (#4041) is required for page adaptations. Wiki captures also include the knowledge toolbar (#4043).

Some before phone controls cannot be reached because of the original layout; those captures show the actual entry page, not a working editor. After captures successfully open the relevant page/panel. Reduced-viewport history is an emulation of available screen height, not proof of a physical soft keyboard. No physical iOS/Android device was tested.

Images are actual browser captures and are kept outside feature PR diffs. Browser tests use explicit mocked application data; they do not exercise production credentials or reprocess production documents.
