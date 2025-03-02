// Multi-Stage Operation Component
// A reusable Alpine.js component for tracking multi-stage operations with progress reporting

document.addEventListener('alpine:init', () => {
    Alpine.data('multiStageOperation', (config = {}) => {
        return {
            // Configurable properties with defaults
            apiEndpoint: config.apiEndpoint || '',
            csrfToken: config.csrfToken || '',
            timeoutDuration: config.timeoutDuration || 15000,
            operationName: config.operationName || 'Operation',
            stageOrder: config.stageOrder || ['Starting'],
            
            // Component state
            isRunning: false,
            results: [],
            currentStage: null,
            
            // Helper methods
            isProgressMessage(message) {
                if (!message) return false;
                const lowerMsg = message.toLowerCase();
                return lowerMsg.includes('running') || 
                       lowerMsg.includes('testing') || 
                       lowerMsg.includes('establishing') || 
                       lowerMsg.includes('initializing') ||
                       lowerMsg.includes('attempting') ||
                       lowerMsg.includes('processing');
            },
            
            // Start the operation
            start(payload = {}, options = {}) {
                const initialStage = options.initialStage || this.stageOrder[0];
                const initialMessage = options.initialMessage || `Initializing ${this.operationName}...`;
                
                this.isRunning = true;
                this.currentStage = initialStage;
                this.results = [{
                    success: true,
                    stage: initialStage,
                    message: initialMessage,
                    state: 'running'
                }];
                
                // Create a timeout promise
                const timeoutPromise = new Promise((_, reject) => {
                    setTimeout(() => reject(new Error(`Operation timeout after ${this.timeoutDuration/1000} seconds`)), this.timeoutDuration);
                });

                if (!this.apiEndpoint) {
                    console.error('No API endpoint specified for the operation');
                    this.results = [{
                        success: false,
                        stage: 'Error',
                        message: 'No API endpoint specified for the operation',
                        state: 'failed'
                    }];
                    this.isRunning = false;
                    return Promise.reject(new Error('No API endpoint specified'));
                }

                // Create the fetch promise
                const fetchPromise = fetch(this.apiEndpoint, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-Token': this.csrfToken
                    },
                    body: JSON.stringify(payload)
                });

                // Race between fetch and timeout
                return Promise.race([fetchPromise, timeoutPromise])
                    .then(response => {
                        if (!response.ok) {
                            throw new Error(`HTTP error! status: ${response.status}`);
                        }
                        
                        const reader = response.body.getReader();
                        const decoder = new TextDecoder();
                        let buffer = '';

                        return new ReadableStream({
                            start: (controller) => {
                                const push = () => {
                                    reader.read().then(({done, value}) => {
                                        if (done) {
                                            controller.close();
                                            return;
                                        }

                                        buffer += decoder.decode(value, {stream: true});
                                        const lines = buffer.split('\n');
                                        buffer = lines.pop(); // Keep the incomplete line

                                        lines.forEach(line => {
                                            if (line.trim()) {
                                                try {
                                                    const result = JSON.parse(line);
                                                    this.currentStage = result.stage;
                                                    
                                                    // Find existing result for this stage
                                                    const existingIndex = this.results.findIndex(r => r.stage === result.stage);
                                                    
                                                    // Determine if this is a progress message
                                                    const isProgress = this.isProgressMessage(result.message);
                                                    
                                                    // Set the state based on the result
                                                    const state = result.state ? result.state :  // Use existing state if provided
                                                        result.error ? 'failed' :
                                                        isProgress ? 'running' :
                                                        result.success ? 'completed' :
                                                        'failed';

                                                    const updatedResult = {
                                                        ...result,
                                                        isProgress: isProgress && !result.error,  // Progress state is false if there's an error
                                                        state,
                                                        success: result.error ? false : result.success
                                                    };
                                                    
                                                    if (existingIndex >= 0) {
                                                        // Update existing result
                                                        this.results[existingIndex] = updatedResult;
                                                    } else {
                                                        // Add new result
                                                        this.results.push(updatedResult);
                                                    }

                                                    // Also update previous stages to completed if this is a new stage
                                                    if (!isProgress && result.success && !result.error) {
                                                        const currentStageIndex = this.stageOrder.indexOf(result.stage);
                                                        this.results.forEach((r, idx) => {
                                                            const stageIndex = this.stageOrder.indexOf(r.stage);
                                                            if (stageIndex < currentStageIndex && r.state === 'running') {
                                                                this.results[idx] = {
                                                                    ...r,
                                                                    state: 'completed',
                                                                    isProgress: false
                                                                };
                                                            }
                                                        });
                                                    }
                                                    
                                                    // Always complete the "Starting Test" stage when any other stage begins
                                                    // This ensures we don't have a stuck "Starting Test" if the first real stage fails
                                                    if (result.stage !== "Starting Test" && !isProgress) {
                                                        const startingTestIndex = this.results.findIndex(r => r.stage === "Starting Test");
                                                        if (startingTestIndex >= 0 && this.results[startingTestIndex].state === "running") {
                                                            this.results[startingTestIndex] = {
                                                                ...this.results[startingTestIndex],
                                                                state: 'completed',
                                                                isProgress: false,
                                                                message: "Initialization complete"
                                                            };
                                                        }
                                                    }
                                                    
                                                    // Sort results according to stage order
                                                    this.results.sort((a, b) => 
                                                        this.stageOrder.indexOf(a.stage) - this.stageOrder.indexOf(b.stage)
                                                    );
                                                } catch (e) {
                                                    console.error('Failed to parse result:', e);
                                                }
                                            }
                                        });

                                        controller.enqueue(value);
                                        push();
                                    }).catch(error => {
                                        controller.error(error);
                                    });
                                };

                                push();
                            }
                        });
                    })
                    .catch(error => {
                        const errorMessage = error.message.includes('timeout')
                            ? `The operation took too long to complete. Please try again.`
                            : `Failed to perform ${this.operationName}`;
                        
                        this.results = [{
                            success: false,
                            stage: 'Error',
                            message: errorMessage,
                            error: error.message,
                            state: 'failed'
                        }];
                        this.currentStage = null;
                        return Promise.reject(error);
                    })
                    .finally(() => {
                        this.isRunning = false;
                        this.currentStage = null;
                    });
            },
            
            // Check if operation was completely successful
            isCompleteSuccess() {
                if (this.results.length === 0 || this.isRunning) return false;
                
                // Every result must be successful
                if (!this.results.every(result => result.success)) return false;
                
                // Must have reached the final stage
                const finalStage = this.stageOrder[this.stageOrder.length - 1];
                return this.results.some(result => result.stage === finalStage);
            },

            // Reset the operation state
            reset() {
                this.isRunning = false;
                this.results = [];
                this.currentStage = null;
            }
        };
    });
}); 