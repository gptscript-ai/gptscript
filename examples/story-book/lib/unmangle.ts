const unmangleStoryName = (storyName: string): string => {
    return storyName.replaceAll('-', ' ').replace(/\b\w/g, c => c.toUpperCase());
}
export default unmangleStoryName;   